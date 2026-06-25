package cluster

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════
// COORDINADOR DEL CLUSTER TCP
// Se conecta a los nodos TCP por red y distribuye
// predicciones usando un pool de conexiones TCP
// ═══════════════════════════════════════════════════════

const (
	timeoutTCP      = 30 * time.Second
	maxConexiones   = 4
	timeoutConexion = 5 * time.Second
)

// ConexionTCP envuelve una conexión TCP con un reader buffered
type ConexionTCP struct {
	conn   net.Conn
	reader *bufio.Reader
}

// PoolConexiones gestiona un pool de conexiones TCP a un nodo
type PoolConexiones struct {
	direccion string
	pool      chan *ConexionTCP
}

// NuevoPool crea un pool de conexiones TCP
func NuevoPool(direccion string, tamano int) *PoolConexiones {
	return &PoolConexiones{
		direccion: direccion,
		pool:      make(chan *ConexionTCP, tamano),
	}
}

// Obtener retorna una conexión del pool o crea una nueva
func (p *PoolConexiones) Obtener() (*ConexionTCP, error) {
	// Intentar obtener una conexión existente del pool
	select {
	case c := <-p.pool:
		return c, nil
	default:
	}

	// No hay conexiones disponibles — crear nueva
	conn, err := net.DialTimeout("tcp", p.direccion, timeoutConexion)
	if err != nil {
		return nil, fmt.Errorf("error conectando a %s: %w", p.direccion, err)
	}
	log.Printf("[Pool] Nueva conexión TCP a %s\n", p.direccion)
	return &ConexionTCP{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, 1024*1024), // 1MB buffer
	}, nil
}

// Devolver retorna una conexión al pool para reutilización
func (p *PoolConexiones) Devolver(c *ConexionTCP) {
	select {
	case p.pool <- c:
	default:
		c.conn.Close() // Pool lleno, cerrar conexión extra
	}
}

// CerrarTodas drena y cierra todas las conexiones del pool
func (p *PoolConexiones) CerrarTodas() {
	for {
		select {
		case c := <-p.pool:
			c.conn.Close()
		default:
			return
		}
	}
}

// ConfigCluster parámetros para inicializar el cluster TCP
type ConfigCluster struct {
	Nodos []ConfigNodo
}

// ConfigNodo describe un nodo TCP del cluster
type ConfigNodo struct {
	Modelo    string // "model1", "model2", "model3"
	Direccion string // "localhost:9001"
}

// Coordinador gestiona las conexiones TCP a los nodos ML
type Coordinador struct {
	pools         map[string]*PoolConexiones
	nodoInfo      map[string]ConfigNodo
	totalPred     int64
	predPorModelo map[string]*int64
	activo        bool
}

// NuevoCoordinador crea el coordinador y verifica la conexión con cada nodo TCP
func NuevoCoordinador(cfg ConfigCluster) (*Coordinador, error) {
	c := &Coordinador{
		pools:         make(map[string]*PoolConexiones),
		nodoInfo:      make(map[string]ConfigNodo),
		predPorModelo: make(map[string]*int64),
		activo:        true,
	}

	for _, nodo := range cfg.Nodos {
		pool := NuevoPool(nodo.Direccion, maxConexiones)
		c.pools[nodo.Modelo] = pool
		c.nodoInfo[nodo.Modelo] = nodo
		counter := int64(0)
		c.predPorModelo[nodo.Modelo] = &counter

		// Verificar conexión TCP con el nodo
		conn, err := pool.Obtener()
		if err != nil {
			return nil, fmt.Errorf("[Coordinador] no se pudo conectar a nodo %s en %s: %w",
				nodo.Modelo, nodo.Direccion, err)
		}
		pool.Devolver(conn)
		log.Printf("[Coordinador] ✔ Conectado a nodo %s en %s\n", nodo.Modelo, nodo.Direccion)
	}

	log.Printf("[Coordinador] ✔ Cluster TCP iniciado — %d nodos conectados\n", len(cfg.Nodos))
	return c, nil
}

// Predecir envía una tarea al nodo TCP correspondiente y espera resultado
func (c *Coordinador) Predecir(modelo string, features []float64) (ResultadoPrediccion, error) {
	pool, ok := c.pools[modelo]
	if !ok {
		return ResultadoPrediccion{}, fmt.Errorf("modelo desconocido: %s", modelo)
	}

	// Obtener conexión TCP del pool
	conn, err := pool.Obtener()
	if err != nil {
		return ResultadoPrediccion{}, fmt.Errorf("error obteniendo conexión TCP: %w", err)
	}

	tareaID := uuid.New().String()
	msg := MensajeTarea{
		TareaID:  tareaID,
		Features: features,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		pool.Devolver(conn)
		return ResultadoPrediccion{}, fmt.Errorf("error serializando tarea: %w", err)
	}

	// Establecer timeout para envío y recepción
	conn.conn.SetDeadline(time.Now().Add(timeoutTCP))

	// Enviar tarea por TCP (JSON + newline como delimitador)
	if _, err := conn.conn.Write(append(data, '\n')); err != nil {
		conn.conn.Close()
		return ResultadoPrediccion{}, fmt.Errorf("error enviando tarea TCP a %s: %w", modelo, err)
	}

	// Leer respuesta del nodo TCP
	linea, err := conn.reader.ReadBytes('\n')
	if err != nil {
		conn.conn.Close()
		return ResultadoPrediccion{}, fmt.Errorf("error leyendo respuesta TCP de %s: %w", modelo, err)
	}

	// Limpiar deadline y devolver conexión al pool
	conn.conn.SetDeadline(time.Time{})
	pool.Devolver(conn)

	var resp MensajeResultado
	if err := json.Unmarshal(linea, &resp); err != nil {
		return ResultadoPrediccion{}, fmt.Errorf("error deserializando respuesta de %s: %w", modelo, err)
	}

	if resp.ErrorMsg != "" {
		return ResultadoPrediccion{}, fmt.Errorf("error del nodo %s: %s", modelo, resp.ErrorMsg)
	}

	// Actualizar contadores atómicos
	atomic.AddInt64(&c.totalPred, 1)
	if counter, ok := c.predPorModelo[modelo]; ok {
		atomic.AddInt64(counter, 1)
	}

	return ResultadoPrediccion{
		TareaID:    resp.TareaID,
		NodoID:     resp.NodoID,
		Modelo:     modelo,
		Resultado:  resp.Resultado,
		DuracionMs: resp.DuracionMs,
	}, nil
}

// EstadoCluster retorna información de cada nodo del cluster
func (c *Coordinador) EstadoCluster() []EstadoNodo {
	estados := make([]EstadoNodo, 0, len(c.predPorModelo))
	for modelo, counter := range c.predPorModelo {
		info := c.nodoInfo[modelo]
		estados = append(estados, EstadoNodo{
			ID:           info.Modelo + "-tcp",
			Modelo:       info.Modelo,
			Activo:       c.activo,
			Predicciones: atomic.LoadInt64(counter),
			UltimaVez:    time.Now(),
		})
	}
	return estados
}

// TotalPredicciones retorna el conteo total del cluster
func (c *Coordinador) TotalPredicciones() int64 {
	return atomic.LoadInt64(&c.totalPred)
}

// Cerrar cierra todas las conexiones TCP del cluster
func (c *Coordinador) Cerrar() {
	c.activo = false
	for _, pool := range c.pools {
		pool.CerrarTodas()
	}
	log.Println("[Coordinador] Cluster TCP detenido")
}
