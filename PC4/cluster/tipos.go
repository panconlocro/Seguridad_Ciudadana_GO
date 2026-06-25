package cluster

import "time"

// ═══════════════════════════════════════════════════════
// TIPOS COMPARTIDOS DEL CLUSTER
// ═══════════════════════════════════════════════════════

// TareaPrediccion es el mensaje que el coordinador envía a un worker
type TareaPrediccion struct {
	ID       string    // identificador único de la tarea
	Modelo   string    // "model1", "model2", "model3"
	Features []float64 // vector de entrada
	Enviada  time.Time // timestamp de envío
}

// ResultadoPrediccion es la respuesta del worker al coordinador
type ResultadoPrediccion struct {
	TareaID    string
	NodoID     string
	Modelo     string
	Resultado  map[string]interface{}
	DuracionMs int64
	Error      error
}

// EstadoNodo representa el estado actual de un worker
type EstadoNodo struct {
	ID           string
	Modelo       string
	Activo       bool
	Predicciones int64
	UltimaVez    time.Time
}
