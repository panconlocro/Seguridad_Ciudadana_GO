package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"securitygo_backend/cluster"
)

// ═══════════════════════════════════════════════════════
// WEBSOCKET HUB — Tiempo real
// Broadcast de predicciones y métricas del cluster
// a todos los clientes conectados
// ═══════════════════════════════════════════════════════

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Permitir cualquier origen (desarrollo)
	},
}

// MensajeWS es el formato de mensaje que se envía por WebSocket
type MensajeWS struct {
	Tipo  string      `json:"tipo"`
	Datos interface{} `json:"datos"`
}

// clienteWS representa un cliente WebSocket conectado
type clienteWS struct {
	conn *websocket.Conn
	mu   sync.Mutex // protege escrituras concurrentes al conn
}

// HubWS gestiona las conexiones WebSocket activas
type HubWS struct {
	mu       sync.RWMutex
	clientes map[*clienteWS]bool
	coord    *cluster.Coordinador
}

// NuevoHubWS crea un hub y arranca la emisión periódica de métricas
func NuevoHubWS(coord *cluster.Coordinador) *HubWS {
	hub := &HubWS{
		clientes: make(map[*clienteWS]bool),
		coord:    coord,
	}
	go hub.emitirMetricas()
	log.Println("[WebSocket] ✔ Hub iniciado")
	return hub
}

// NumClientes retorna la cantidad de clientes conectados
func (h *HubWS) NumClientes() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clientes)
}

// Broadcast envía un mensaje a todos los clientes conectados
func (h *HubWS) Broadcast(msg MensajeWS) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WebSocket] error serializando: %v\n", err)
		return
	}

	// Copiar lista de clientes para no mantener el lock durante writes
	h.mu.RLock()
	clientes := make([]*clienteWS, 0, len(h.clientes))
	for c := range h.clientes {
		clientes = append(clientes, c)
	}
	h.mu.RUnlock()

	for _, c := range clientes {
		c.mu.Lock()
		err := c.conn.WriteMessage(websocket.TextMessage, data)
		c.mu.Unlock()
		if err != nil {
			log.Printf("[WebSocket] error enviando a cliente: %v\n", err)
			// No cerramos aquí — el reader goroutine se encargará
		}
	}
}

// HandleWS maneja la conexión WebSocket (upgrade HTTP → WS)
func (h *HubWS) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] error upgrade: %v\n", err)
		return
	}

	cliente := &clienteWS{conn: conn}

	h.mu.Lock()
	h.clientes[cliente] = true
	total := len(h.clientes)
	h.mu.Unlock()

	log.Printf("[WebSocket] Cliente conectado — total: %d\n", total)

	// Enviar mensaje de bienvenida
	bienvenida := MensajeWS{
		Tipo: "conexion",
		Datos: map[string]interface{}{
			"mensaje":   "Conectado a SecurityGO WebSocket",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}
	data, _ := json.Marshal(bienvenida)
	conn.WriteMessage(websocket.TextMessage, data)

	// Bloquear leyendo mensajes — mantiene la conexión viva
	// Cuando el cliente se desconecta, ReadMessage retorna error
	defer func() {
		h.mu.Lock()
		delete(h.clientes, cliente)
		numClientes := len(h.clientes)
		h.mu.Unlock()
		conn.Close()
		log.Printf("[WebSocket] Cliente desconectado — total: %d\n", numClientes)
	}()

	conn.SetReadLimit(512)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// emitirMetricas envía métricas del cluster cada 5 segundos
func (h *HubWS) emitirMetricas() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if h.NumClientes() == 0 {
			continue
		}

		estados := h.coord.EstadoCluster()
		nodos := make([]map[string]interface{}, len(estados))
		for i, e := range estados {
			nodos[i] = map[string]interface{}{
				"id":           e.ID,
				"modelo":       e.Modelo,
				"activo":       e.Activo,
				"predicciones": e.Predicciones,
			}
		}

		msg := MensajeWS{
			Tipo: "metricas",
			Datos: map[string]interface{}{
				"nodos":                nodos,
				"predicciones_totales": h.coord.TotalPredicciones(),
				"clientes_ws":         h.NumClientes(),
				"timestamp":           time.Now().UTC().Format(time.RFC3339),
			},
		}

		h.Broadcast(msg)
	}
}
