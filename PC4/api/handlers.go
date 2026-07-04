package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"securitygo_pc4/cluster"
	"securitygo_pc4/db"
)

// ═══════════════════════════════════════════════════════
// HANDLERS DE LA API REST
// ═══════════════════════════════════════════════════════

// Handler agrupa coordinador, bases de datos y WebSocket hub
type Handler struct {
	coord *cluster.Coordinador
	mongo *db.ClienteMongo
	redis *db.ClienteRedis // puede ser nil si Redis no está disponible
	hub   *HubWS
}

// NuevoHandler crea los handlers con sus dependencias
func NuevoHandler(coord *cluster.Coordinador, mongo *db.ClienteMongo, redis *db.ClienteRedis, hub *HubWS) *Handler {
	return &Handler{coord: coord, mongo: mongo, redis: redis, hub: hub}
}

// -- Request / Response types --

type RespuestaBase struct {
	OK        bool        `json:"ok"`
	Timestamp string      `json:"timestamp"`
	Datos     interface{} `json:"datos,omitempty"`
	Error     string      `json:"error,omitempty"`
}

type RequestModel1 struct {
	Hour             int `json:"hour"`
	DayOfWeek        int `json:"day_of_week"`
	Month            int `json:"month"`
	Area             int `json:"area"`
	PremisCd         int `json:"premis_cd"`
	Part12           int `json:"part_1_2"`
	VictimIdentified int `json:"victim_identified"`
	DaysToReport     int `json:"days_to_report"`
}

type RequestModel2 struct {
	Hour      int `json:"hour"`
	DayOfWeek int `json:"day_of_week"`
	Month     int `json:"month"`
	CrmCd     int `json:"crm_cd"`
	PremisCd  int `json:"premis_cd"`
	Part12    int `json:"part_1_2"`
	Area      int `json:"area"`
}

type RequestModel3 struct {
	CrmCd            int `json:"crm_cd"`
	Area             int `json:"area"`
	Hour             int `json:"hour"`
	DayOfWeek        int `json:"day_of_week"`
	PremisCd         int `json:"premis_cd"`
	WeaponPresent    int `json:"weapon_present"`
	VictimIdentified int `json:"victim_identified"`
	DaysToReport     int `json:"days_to_report"`
	Part12           int `json:"part_1_2"`
}

// -- Helpers --

func respOK(w http.ResponseWriter, datos interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RespuestaBase{
		OK:        true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Datos:     datos,
	})
}

func respError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(RespuestaBase{
		OK:        false,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Error:     msg,
	})
}

func (h *Handler) guardarEnMongo(modelo, nodo string, features map[string]interface{}, resultado map[string]interface{}, duracionMs int64) {
	reg := db.RegistroPrediccion{
		Modelo:     modelo,
		NodoWorker: nodo,
		Features:   features,
		Resultado:  resultado,
		DuracionMs: duracionMs,
	}
	if err := h.mongo.GuardarPrediccion(reg); err != nil {
		log.Printf("[API] advertencia: no se pudo guardar en MongoDB: %v\n", err)
	}
}

// broadcastPrediccion notifica a clientes WebSocket sobre una nueva predicción
func (h *Handler) broadcastPrediccion(modelo, nodo string, resultado map[string]interface{}, duracionMs int64, desdeCache bool) {
	if h.hub == nil {
		return
	}
	h.hub.Broadcast(MensajeWS{
		Tipo: "prediccion",
		Datos: map[string]interface{}{
			"modelo":      modelo,
			"nodo":        nodo,
			"resultado":   resultado,
			"duracion_ms": duracionMs,
			"desde_cache": desdeCache,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// -- Endpoints --

// PredecirTipoCrimen POST /predict/crime-type
// Features: hour, day_of_week, month, area, premis_cd, part_1_2, victim_identified, days_to_report
func (h *Handler) PredecirTipoCrimen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respError(w, http.StatusMethodNotAllowed, "usar POST")
		return
	}
	var req RequestModel1
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respError(w, http.StatusBadRequest, fmt.Sprintf("JSON inválido: %v", err))
		return
	}

	features := []float64{
		float64(req.Hour), float64(req.DayOfWeek), float64(req.Month),
		float64(req.Area), float64(req.PremisCd), float64(req.Part12),
		float64(req.VictimIdentified), float64(req.DaysToReport),
	}

	// Verificar caché Redis
	if h.redis != nil {
		if cached, ok := h.redis.ObtenerCache("model1", features); ok {
			respOK(w, map[string]interface{}{
				"modelo":      "model1",
				"prediccion":  cached,
				"desde_cache": true,
				"duracion_ms": 0,
			})
			go h.broadcastPrediccion("model1", "redis-cache", cached, 0, true)
			return
		}
	}

	log.Printf("[API] /predict/crime-type — features: %v\n", features)
	inicio := time.Now()
	resultado, err := h.coord.Predecir("model1", features)
	if err != nil {
		respError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respOK(w, map[string]interface{}{
		"modelo":      "model1",
		"nodo_worker": resultado.NodoID,
		"prediccion":  resultado.Resultado,
		"duracion_ms": resultado.DuracionMs,
		"desde_cache": false,
	})

	featureMap := map[string]interface{}{
		"hour": req.Hour, "day_of_week": req.DayOfWeek, "month": req.Month,
		"area": req.Area, "premis_cd": req.PremisCd, "part_1_2": req.Part12,
		"victim_identified": req.VictimIdentified, "days_to_report": req.DaysToReport,
	}
	duracion := time.Since(inicio).Milliseconds()
	go h.guardarEnMongo("model1", resultado.NodoID, featureMap, resultado.Resultado, duracion)
	if h.redis != nil {
		go h.redis.CachearPrediccion("model1", features, resultado.Resultado)
	}
	go h.broadcastPrediccion("model1", resultado.NodoID, resultado.Resultado, resultado.DuracionMs, false)
}

// PredecirZonaRiesgo POST /predict/risk-zone
// Features: hour, day_of_week, month, crm_cd, premis_cd, part_1_2, area
func (h *Handler) PredecirZonaRiesgo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respError(w, http.StatusMethodNotAllowed, "usar POST")
		return
	}
	var req RequestModel2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respError(w, http.StatusBadRequest, fmt.Sprintf("JSON inválido: %v", err))
		return
	}

	features := []float64{
		float64(req.Hour), float64(req.DayOfWeek), float64(req.Month),
		float64(req.CrmCd), float64(req.PremisCd), float64(req.Part12),
		float64(req.Area),
	}

	// Verificar caché Redis
	if h.redis != nil {
		if cached, ok := h.redis.ObtenerCache("model2", features); ok {
			respOK(w, map[string]interface{}{
				"modelo":      "model2",
				"prediccion":  cached,
				"desde_cache": true,
				"duracion_ms": 0,
			})
			go h.broadcastPrediccion("model2", "redis-cache", cached, 0, true)
			return
		}
	}

	log.Printf("[API] /predict/risk-zone — features: %v\n", features)
	inicio := time.Now()
	resultado, err := h.coord.Predecir("model2", features)
	if err != nil {
		respError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respOK(w, map[string]interface{}{
		"modelo":      "model2",
		"nodo_worker": resultado.NodoID,
		"prediccion":  resultado.Resultado,
		"duracion_ms": resultado.DuracionMs,
		"desde_cache": false,
	})

	featureMap := map[string]interface{}{
		"hour": req.Hour, "day_of_week": req.DayOfWeek, "month": req.Month,
		"crm_cd": req.CrmCd, "premis_cd": req.PremisCd, "part_1_2": req.Part12,
		"area": req.Area,
	}
	duracion := time.Since(inicio).Milliseconds()
	go h.guardarEnMongo("model2", resultado.NodoID, featureMap, resultado.Resultado, duracion)
	if h.redis != nil {
		go h.redis.CachearPrediccion("model2", features, resultado.Resultado)
	}
	go h.broadcastPrediccion("model2", resultado.NodoID, resultado.Resultado, resultado.DuracionMs, false)
}

// PredecirProbArresto POST /predict/arrest-prob
// Features: crm_cd, area, hour, day_of_week, premis_cd, weapon_present, victim_identified, days_to_report, part_1_2
func (h *Handler) PredecirProbArresto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respError(w, http.StatusMethodNotAllowed, "usar POST")
		return
	}
	var req RequestModel3
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respError(w, http.StatusBadRequest, fmt.Sprintf("JSON inválido: %v", err))
		return
	}

	features := []float64{
		float64(req.CrmCd), float64(req.Area), float64(req.Hour),
		float64(req.DayOfWeek), float64(req.PremisCd), float64(req.WeaponPresent),
		float64(req.VictimIdentified), float64(req.DaysToReport), float64(req.Part12),
	}

	// Verificar caché Redis
	if h.redis != nil {
		if cached, ok := h.redis.ObtenerCache("model3", features); ok {
			respOK(w, map[string]interface{}{
				"modelo":      "model3",
				"prediccion":  cached,
				"desde_cache": true,
				"duracion_ms": 0,
			})
			go h.broadcastPrediccion("model3", "redis-cache", cached, 0, true)
			return
		}
	}

	log.Printf("[API] /predict/arrest-prob — features: %v\n", features)
	inicio := time.Now()
	resultado, err := h.coord.Predecir("model3", features)
	if err != nil {
		respError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respOK(w, map[string]interface{}{
		"modelo":      "model3",
		"nodo_worker": resultado.NodoID,
		"prediccion":  resultado.Resultado,
		"duracion_ms": resultado.DuracionMs,
		"desde_cache": false,
	})

	featureMap := map[string]interface{}{
		"crm_cd": req.CrmCd, "area": req.Area, "hour": req.Hour,
		"day_of_week": req.DayOfWeek, "premis_cd": req.PremisCd,
		"weapon_present": req.WeaponPresent, "victim_identified": req.VictimIdentified,
		"days_to_report": req.DaysToReport, "part_1_2": req.Part12,
	}
	duracion := time.Since(inicio).Milliseconds()
	go h.guardarEnMongo("model3", resultado.NodoID, featureMap, resultado.Resultado, duracion)
	if h.redis != nil {
		go h.redis.CachearPrediccion("model3", features, resultado.Resultado)
	}
	go h.broadcastPrediccion("model3", resultado.NodoID, resultado.Resultado, resultado.DuracionMs, false)
}

// HealthCheck GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	estados := h.coord.EstadoCluster()
	total, _ := h.mongo.ContarPredicciones()

	nodos := make([]map[string]interface{}, len(estados))
	for i, e := range estados {
		nodos[i] = map[string]interface{}{
			"id":           e.ID,
			"modelo":       e.Modelo,
			"activo":       e.Activo,
			"predicciones": e.Predicciones,
		}
	}

	datos := map[string]interface{}{
		"estado":             "OK",
		"nodos":              nodos,
		"total_pred_cluster": h.coord.TotalPredicciones(),
		"total_pred_mongodb": total,
	}

	// Agregar stats de Redis si está disponible
	if h.redis != nil {
		datos["redis"] = h.redis.EstadisticasCache()
	}

	// Agregar stats de WebSocket
	if h.hub != nil {
		datos["websocket_clientes"] = h.hub.NumClientes()
	}

	respOK(w, datos)
}

// HistorialPredicciones GET /predictions?model=model1&limit=10
func (h *Handler) HistorialPredicciones(w http.ResponseWriter, r *http.Request) {
	modelo := r.URL.Query().Get("model")
	limiteStr := r.URL.Query().Get("limit")
	limite := 10
	if limiteStr != "" {
		if v, err := strconv.Atoi(limiteStr); err == nil && v > 0 {
			limite = v
		}
	}
	if limite > 100 {
		limite = 100
	}

	registros, err := h.mongo.ObtenerPredicciones(modelo, limite)
	if err != nil {
		respError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respOK(w, map[string]interface{}{
		"total":     len(registros),
		"registros": registros,
	})
}

// EstadisticasCache GET /cache/stats
func (h *Handler) EstadisticasCache(w http.ResponseWriter, r *http.Request) {
	if h.redis == nil {
		respOK(w, map[string]interface{}{
			"estado":  "no_disponible",
			"mensaje": "Redis no está configurado",
		})
		return
	}

	respOK(w, h.redis.EstadisticasCache())
}

// -- Middleware de logging --

func LoggingMiddleware(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		siguiente.ServeHTTP(w, r)
		log.Printf("[API] %s %s — %v\n", r.Method, r.URL.Path, time.Since(inicio).Round(time.Millisecond))
	})
}

// -- CORS para pruebas locales --

func CORSMiddleware(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if strings.ToUpper(r.Method) == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		siguiente.ServeHTTP(w, r)
	})
}
