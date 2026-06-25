package cluster

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════
// COORDINADOR DEL CLUSTER
// Recibe peticiones de la API y las distribuye a workers
// usando channels — un channel de tareas por modelo
// ═══════════════════════════════════════════════════════

const (
	bufferCanal  = 50 // tamaño del buffer de tareas por modelo
	timeoutTarea = 30 * time.Second
)

// Coordinador gestiona los workers y enruta predicciones
type Coordinador struct {
	// Un channel de tareas por cada modelo
	canalModel1 chan TareaPrediccion
	canalModel2 chan TareaPrediccion
	canalModel3 chan TareaPrediccion

	// Channel único de resultados (todos los workers escriben aquí)
	resultados chan ResultadoPrediccion

	// Mapa de resultados pendientes: tareaID → channel de respuesta
	mu      sync.Mutex
	pending map[string]chan ResultadoPrediccion

	// Workers registrados
	workers []*NodoWorker

	// Contador total de predicciones del cluster
	totalPred int64

	activo bool
}

// ConfigCluster parámetros para inicializar el cluster
type ConfigCluster struct {
	RutaModel1     string
	RutaModel2     string
	RutaModel3     string
	WorkersPorNodo int // cuántos workers por modelo (mínimo 1)
}

// NuevoCoordinador crea el coordinador e inicia todos los workers
func NuevoCoordinador(cfg ConfigCluster) (*Coordinador, error) {
	if cfg.WorkersPorNodo < 1 {
		cfg.WorkersPorNodo = 1
	}

	c := &Coordinador{
		canalModel1: make(chan TareaPrediccion, bufferCanal),
		canalModel2: make(chan TareaPrediccion, bufferCanal),
		canalModel3: make(chan TareaPrediccion, bufferCanal),
		resultados:  make(chan ResultadoPrediccion, bufferCanal*3),
		pending:     make(map[string]chan ResultadoPrediccion),
		activo:      true,
	}

	modelos := []struct {
		tipo  string
		ruta  string
		canal chan TareaPrediccion
	}{
		{"model1", cfg.RutaModel1, c.canalModel1},
		{"model2", cfg.RutaModel2, c.canalModel2},
		{"model3", cfg.RutaModel3, c.canalModel3},
	}

	for _, m := range modelos {
		for i := 1; i <= cfg.WorkersPorNodo; i++ {
			id := fmt.Sprintf("%s-worker-%d", m.tipo, i)
			w, err := NuevoNodoWorker(id, m.ruta, m.canal, c.resultados)
			if err != nil {
				return nil, fmt.Errorf("[Coordinador] no se pudo iniciar %s: %w", id, err)
			}
			w.Iniciar()
			c.workers = append(c.workers, w)
		}
	}

	// Goroutine que despacha resultados a quienes los esperan
	go c.despachadorResultados()

	log.Printf("[Coordinador] ✔ Cluster iniciado — %d workers activos (%d por modelo)\n",
		len(c.workers), cfg.WorkersPorNodo)
	return c, nil
}

// Predecir envía una tarea al worker correcto y espera el resultado
func (c *Coordinador) Predecir(modelo string, features []float64) (ResultadoPrediccion, error) {
	tareaID := uuid.New().String()
	respCh := make(chan ResultadoPrediccion, 1)

	c.mu.Lock()
	c.pending[tareaID] = respCh
	c.mu.Unlock()

	tarea := TareaPrediccion{
		ID:       tareaID,
		Modelo:   modelo,
		Features: features,
		Enviada:  time.Now(),
	}

	// Enrutar al canal del modelo correspondiente
	var canal chan TareaPrediccion
	switch modelo {
	case "model1":
		canal = c.canalModel1
	case "model2":
		canal = c.canalModel2
	case "model3":
		canal = c.canalModel3
	default:
		c.mu.Lock()
		delete(c.pending, tareaID)
		c.mu.Unlock()
		return ResultadoPrediccion{}, fmt.Errorf("modelo desconocido: %s", modelo)
	}

	select {
	case canal <- tarea:
	case <-time.After(timeoutTarea):
		c.mu.Lock()
		delete(c.pending, tareaID)
		c.mu.Unlock()
		return ResultadoPrediccion{}, fmt.Errorf("timeout enviando tarea al cluster")
	}

	// Esperar respuesta del worker
	select {
	case resultado := <-respCh:
		atomic.AddInt64(&c.totalPred, 1)
		return resultado, resultado.Error
	case <-time.After(timeoutTarea):
		c.mu.Lock()
		delete(c.pending, tareaID)
		c.mu.Unlock()
		return ResultadoPrediccion{}, fmt.Errorf("timeout esperando predicción")
	}
}

// EstadoCluster retorna información de todos los workers
func (c *Coordinador) EstadoCluster() []EstadoNodo {
	estados := make([]EstadoNodo, len(c.workers))
	for i, w := range c.workers {
		estados[i] = w.Estado()
	}
	return estados
}

// TotalPredicciones retorna el conteo total del cluster
func (c *Coordinador) TotalPredicciones() int64 {
	return atomic.LoadInt64(&c.totalPred)
}

// Cerrar cierra todos los canales del cluster
func (c *Coordinador) Cerrar() {
	c.activo = false
	close(c.canalModel1)
	close(c.canalModel2)
	close(c.canalModel3)
	log.Println("[Coordinador] Cluster detenido")
}

// despachadorResultados lee el canal de resultados y los entrega al solicitante
func (c *Coordinador) despachadorResultados() {
	for resultado := range c.resultados {
		c.mu.Lock()
		ch, ok := c.pending[resultado.TareaID]
		if ok {
			delete(c.pending, resultado.TareaID)
		}
		c.mu.Unlock()

		if ok {
			ch <- resultado
		}
	}
}
