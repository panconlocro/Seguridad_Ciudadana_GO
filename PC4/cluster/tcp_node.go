package cluster

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════
// NODO TCP — Servidor TCP para un modelo ML
// Cada nodo escucha en un puerto, recibe tareas JSON
// y responde con predicciones vía TCP
// ═══════════════════════════════════════════════════════

// NodoTCP representa un nodo del cluster que escucha por TCP
type NodoTCP struct {
	id         string
	puerto     string
	modelo     *ModeloJSON
	modeloTipo string
	listener   net.Listener
	contador   int64
	activo     bool
}

// NuevoNodoTCP crea un nodo TCP que carga un modelo ML
func NuevoNodoTCP(id, puerto, rutaModelo string) (*NodoTCP, error) {
	modelo, err := cargarModeloJSON(rutaModelo)
	if err != nil {
		return nil, fmt.Errorf("[NodoTCP %s] error cargando modelo: %w", id, err)
	}

	log.Printf("[NodoTCP %s] ✔ Modelo %s cargado desde %s\n", id, modelo.Tipo, rutaModelo)

	return &NodoTCP{
		id:         id,
		puerto:     puerto,
		modelo:     modelo,
		modeloTipo: modelo.Tipo,
		activo:     true,
	}, nil
}

// Iniciar abre el listener TCP y empieza a aceptar conexiones
func (n *NodoTCP) Iniciar() error {
	listener, err := net.Listen("tcp", ":"+n.puerto)
	if err != nil {
		return fmt.Errorf("[NodoTCP %s] error escuchando en :%s: %w", n.id, n.puerto, err)
	}
	n.listener = listener

	log.Printf("[NodoTCP %s] ✔ Escuchando TCP en :%s\n", n.id, n.puerto)

	go func() {
		for {
			conn, err := n.listener.Accept()
			if err != nil {
				if n.activo {
					log.Printf("[NodoTCP %s] error accept: %v\n", n.id, err)
				}
				return
			}
			go n.manejarConexion(conn)
		}
	}()

	return nil
}

// manejarConexion procesa tareas de una conexión TCP persistente
// Protocolo: cada mensaje es un JSON terminado en \n
func (n *NodoTCP) manejarConexion(conn net.Conn) {
	defer conn.Close()
	remoto := conn.RemoteAddr().String()
	log.Printf("[NodoTCP %s] Conexión aceptada desde %s\n", n.id, remoto)

	reader := bufio.NewReaderSize(conn, 1024*1024) // 1MB buffer

	for {
		linea, err := reader.ReadBytes('\n')
		if err != nil {
			log.Printf("[NodoTCP %s] Conexión cerrada con %s: %v\n", n.id, remoto, err)
			return
		}

		var msg MensajeTarea
		if err := json.Unmarshal(linea, &msg); err != nil {
			resp := MensajeResultado{
				NodoID:   n.id,
				ErrorMsg: fmt.Sprintf("JSON inválido: %v", err),
			}
			data, _ := json.Marshal(resp)
			conn.Write(append(data, '\n'))
			continue
		}

		inicio := time.Now()
		resultado, inferErr := inferirConModelo(n.modelo, msg.Features)
		duracion := time.Since(inicio).Milliseconds()

		resp := MensajeResultado{
			TareaID:    msg.TareaID,
			NodoID:     n.id,
			Resultado:  resultado,
			DuracionMs: duracion,
		}
		if inferErr != nil {
			resp.ErrorMsg = inferErr.Error()
		}

		atomic.AddInt64(&n.contador, 1)

		data, _ := json.Marshal(resp)
		conn.Write(append(data, '\n'))
	}
}

// Estado retorna el estado actual del nodo TCP
func (n *NodoTCP) Estado() EstadoNodo {
	return EstadoNodo{
		ID:           n.id,
		Modelo:       n.modeloTipo,
		Activo:       n.activo,
		Predicciones: atomic.LoadInt64(&n.contador),
		UltimaVez:    time.Now(),
	}
}

// Cerrar detiene el nodo TCP
func (n *NodoTCP) Cerrar() {
	n.activo = false
	if n.listener != nil {
		n.listener.Close()
	}
	log.Printf("[NodoTCP %s] Detenido — total predicciones: %d\n",
		n.id, atomic.LoadInt64(&n.contador))
}
