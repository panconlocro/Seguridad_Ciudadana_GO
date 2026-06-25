package cluster

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════
// NODO WORKER
// Cada worker carga un modelo JSON y responde predicciones
// recibidas desde el coordinador vía channel
// ═══════════════════════════════════════════════════════

// -- Estructuras mínimas para deserializar los modelos JSON --

type NodoArbol struct {
	EsHoja     bool           `json:"EsHoja"`
	ClaseHoja  string         `json:"ClaseHoja"`
	ValorHoja  float64        `json:"ValorHoja"`
	ConteoHoja map[string]int `json:"ConteoHoja"`
	FeatureIdx int            `json:"FeatureIdx"`
	Umbral     float64        `json:"Umbral"`
	Izquierda  *NodoArbol     `json:"Izquierda"`
	Derecha    *NodoArbol     `json:"Derecha"`
}

type ModeloJSON struct {
	Tipo     string   `json:"Tipo"`
	Features []string `json:"Features"`
	Modelo1  *struct {
		Arboles    []*NodoArbol `json:"Arboles"`
		NumArboles int          `json:"NumArboles"`
	} `json:"Modelo1"`
	Modelo2 *struct {
		ArbolesLat []*NodoArbol `json:"ArbolesLat"`
		ArbolesLon []*NodoArbol `json:"ArbolesLon"`
		NumArboles int          `json:"NumArboles"`
	} `json:"Modelo2"`
	Modelo3 *struct {
		Arboles    []*NodoArbol `json:"Arboles"`
		NumArboles int          `json:"NumArboles"`
	} `json:"Modelo3"`
}

// -- Funciones de inferencia sobre el árbol deserializado --

func predecirClasificacion(nodo *NodoArbol, features []float64) string {
	if nodo == nil {
		return ""
	}
	if nodo.EsHoja {
		return nodo.ClaseHoja
	}
	if nodo.FeatureIdx < 0 || nodo.FeatureIdx >= len(features) {
		return ""
	}
	if features[nodo.FeatureIdx] <= nodo.Umbral {
		return predecirClasificacion(nodo.Izquierda, features)
	}
	return predecirClasificacion(nodo.Derecha, features)
}

func predecirRegresion(nodo *NodoArbol, features []float64) float64 {
	if nodo == nil {
		return 0
	}
	if nodo.EsHoja {
		return nodo.ValorHoja
	}
	if nodo.FeatureIdx < 0 || nodo.FeatureIdx >= len(features) {
		return 0
	}
	if features[nodo.FeatureIdx] <= nodo.Umbral {
		return predecirRegresion(nodo.Izquierda, features)
	}
	return predecirRegresion(nodo.Derecha, features)
}

// -- Worker --

// NodoWorker representa un nodo del cluster que procesa predicciones
type NodoWorker struct {
	id         string
	modeloTipo string
	modelo     *ModeloJSON
	tareas     <-chan TareaPrediccion
	resultados chan<- ResultadoPrediccion
	contador   int64
}

// NuevoNodoWorker crea un worker, carga el modelo y empieza a escuchar
func NuevoNodoWorker(
	id string,
	rutaModelo string,
	tareas <-chan TareaPrediccion,
	resultados chan<- ResultadoPrediccion,
) (*NodoWorker, error) {
	modelo, err := cargarModeloJSON(rutaModelo)
	if err != nil {
		return nil, fmt.Errorf("[Worker %s] error cargando modelo: %w", id, err)
	}
	log.Printf("[Worker %s] ✔ Modelo %s cargado desde %s\n", id, modelo.Tipo, rutaModelo)

	w := &NodoWorker{
		id:         id,
		modeloTipo: modelo.Tipo,
		modelo:     modelo,
		tareas:     tareas,
		resultados: resultados,
	}
	return w, nil
}

// Iniciar arranca la goroutine del worker
func (w *NodoWorker) Iniciar() {
	go func() {
		log.Printf("[Worker %s] Iniciado — esperando tareas de %s\n", w.id, w.modeloTipo)
		for tarea := range w.tareas {
			inicio := time.Now()
			resultado := w.procesarTarea(tarea)
			resultado.DuracionMs = time.Since(inicio).Milliseconds()
			atomic.AddInt64(&w.contador, 1)
			w.resultados <- resultado
		}
		log.Printf("[Worker %s] Canal cerrado — total predicciones: %d\n", w.id, w.contador)
	}()
}

// Estado retorna el estado actual del worker
func (w *NodoWorker) Estado() EstadoNodo {
	return EstadoNodo{
		ID:           w.id,
		Modelo:       w.modeloTipo,
		Activo:       true,
		Predicciones: atomic.LoadInt64(&w.contador),
		UltimaVez:    time.Now(),
	}
}

func (w *NodoWorker) procesarTarea(t TareaPrediccion) ResultadoPrediccion {
	res := ResultadoPrediccion{
		TareaID: t.ID,
		NodoID:  w.id,
		Modelo:  w.modeloTipo,
	}

	switch w.modeloTipo {
	case "model1":
		if w.modelo.Modelo1 == nil {
			res.Error = fmt.Errorf("modelo1 no cargado")
			return res
		}
		votos := make(map[string]int)
		for _, arbol := range w.modelo.Modelo1.Arboles {
			votos[predecirClasificacion(arbol, t.Features)]++
		}
		mejor, maxV := "", 0
		for clase, v := range votos {
			if v > maxV {
				maxV, mejor = v, clase
			}
		}
		confianza := float64(maxV) / float64(w.modelo.Modelo1.NumArboles) * 100
		res.Resultado = map[string]interface{}{
			"tipo_crimen": mejor,
			"confianza":   fmt.Sprintf("%.2f%%", confianza),
		}

	case "model2":
		if w.modelo.Modelo2 == nil {
			res.Error = fmt.Errorf("modelo2 no cargado")
			return res
		}
		sumLat, sumLon := 0.0, 0.0
		n := float64(w.modelo.Modelo2.NumArboles)
		for i := 0; i < w.modelo.Modelo2.NumArboles; i++ {
			sumLat += predecirRegresion(w.modelo.Modelo2.ArbolesLat[i], t.Features)
			sumLon += predecirRegresion(w.modelo.Modelo2.ArbolesLon[i], t.Features)
		}
		lat, lon := sumLat/n, sumLon/n
		res.Resultado = map[string]interface{}{
			"latitud":   lat,
			"longitud":  lon,
			"gmaps_url": fmt.Sprintf("https://maps.google.com/?q=%.6f,%.6f", lat, lon),
		}

	case "model3":
		if w.modelo.Modelo3 == nil {
			res.Error = fmt.Errorf("modelo3 no cargado")
			return res
		}
		votosArresto := 0
		for _, arbol := range w.modelo.Modelo3.Arboles {
			if predecirClasificacion(arbol, t.Features) == "ARRESTO" {
				votosArresto++
			}
		}
		prob := float64(votosArresto) / float64(w.modelo.Modelo3.NumArboles)
		clase := "NO_ARRESTO"
		if prob >= 0.5 {
			clase = "ARRESTO"
		}
		res.Resultado = map[string]interface{}{
			"prediccion":           clase,
			"probabilidad_arresto": fmt.Sprintf("%.2f%%", prob*100),
		}

	default:
		res.Error = fmt.Errorf("tipo de modelo desconocido: %s", w.modeloTipo)
	}
	return res
}

func cargarModeloJSON(ruta string) (*ModeloJSON, error) {
	data, err := os.ReadFile(ruta)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer %q: %w", ruta, err)
	}
	var modelo ModeloJSON
	if err := json.Unmarshal(data, &modelo); err != nil {
		return nil, fmt.Errorf("JSON inválido en %q: %w", ruta, err)
	}
	return &modelo, nil
}
