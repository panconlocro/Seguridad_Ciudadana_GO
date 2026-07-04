1ACC0065 - Programación Concurrente y Distribuida
Informe de Práctica Calificada 4
Carrera de Ciencias de la Computación
Docente: Carlos Alberto Jara García
Nombres y Apellidos Código
Rosa María Rodríguez Valencia U202212675
Mildred Micaela Marchan Quispe U202213292
Nicolas Francisco Miranda Rafael U202216895
2026 - 1

---

## 1. Distribución (Cluster de nodos ML mediante TCP)

Se ha implementado una arquitectura fuertemente distribuida basada en el patrón Coordinador-Worker. A diferencia de implementaciones de un solo proceso, el sistema aísla los componentes de red y los cálculos matemáticos pesados en nodos independientes que se comunican mediante protocolos TCP internos. El objetivo es permitir el procesamiento verdaderamente concurrente y escalable de solicitudes de predicción de gran volumen.

### Componentes del cluster

El clúster está compuesto por dos abstracciones principales: el Coordinador TCP y los Nodos TCP (Workers).

**El Coordinador TCP (`coordinator.go`)** actúa como el API Gateway interno. Se encarga de recibir las tareas de los handlers HTTP, identificar a qué modelo corresponde y despacharlas hacia el nodo adecuado. En lugar de crear y destruir conexiones repetitivamente, el coordinador implementa un patrón de **Connection Pooling** (pool de conexiones) que mantiene sockets TCP persistentes y pre-calentados hacia cada nodo, reduciendo significativamente la sobrecarga de handshake de la red bajo alto tráfico:

```go
// NuevoPool crea un pool de conexiones TCP persistentes concurrentes
func NuevoPool(direccion string, tamano int) *PoolConexiones {
	return &PoolConexiones{
		direccion: direccion,
		pool:      make(chan *ConexionTCP, tamano),
	}
}
```

**Los Nodos TCP (`tcp_node.go`)** son microservicios que operan de forma estrictamente independiente. Al iniciar, cada nodo carga su modelo entrenado en memoria RAM (reconstruyendo la estructura del bosque *Random Forest*) e inicia un `net.Listener` en un puerto específico. Al aceptar una conexión TCP, leen mensajes codificados en JSON (delimitados por saltos de línea `\n`) de manera asíncrona usando `bufio.Reader`, ejecutan la inferencia de manera aislada y devuelven el resultado al coordinador:

```go
// manejarConexion procesa tareas de una conexión TCP persistente
func (n *NodoTCP) manejarConexion(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReaderSize(conn, 1024*1024) 
	for {
		linea, err := reader.ReadBytes('\n')
		var msg MensajeTarea
		json.Unmarshal(linea, &msg)
        
		// Ejecuta la inferencia matemáticamente pesada
		resultado, _ := inferirConModelo(n.modelo, msg.Features)
        
        // ... responde al coordinador TCP
	}
}
```

### Configuración de la topología

La red interna del cluster está segmentada por puertos TCP exclusivos, resultando en un ecosistema de procesamiento distribuido altamente cohesivo y modular:

| Nodo Servidor | Puerto TCP | Función Principal |
| ------------- | ---------- | ----------------- |
| `nodo-model1` | `:9001` | Clasificación tipo de crimen |
| `nodo-model2` | `:9002` | Predicción de coordenadas de zona de riesgo |
| `nodo-model3` | `:9003` | Estimación de probabilidad de arresto |

### Mecanismos de concurrencia utilizados
Se utilizaron los mecanismos nativos de Go (Goroutines y Channels) aplicados de manera óptima al diseño de red:
1.  **Goroutines Desacopladas**: Cada cliente TCP que es aceptado por un nodo de Machine Learning levanta inmediatamente su propia goroutine independiente (`go n.manejarConexion(conn)`). Esto permite a un mismo nodo atender a múltiples peticiones simultáneas sin bloquearse esperando resoluciones de disco o red.
2.  **Operaciones Atómicas (`sync/atomic`)**: Para mantener la observabilidad del sistema en tiempo real, los contadores de las predicciones totales completadas por nodo se gestionan con la librería `atomic`. Esto asegura que el estado global se actualice con precisión milimétrica bajo máxima concurrencia, sin usar semáforos (`Mutex`) que generen cuellos de botella.
3.  **Channels como Colas de Red**: Los *channels* (`chan`) ya no se usan para pasar datos en memoria a los workers como en entregas anteriores. Ahora se utilizan arquitectónicamente para construir el **Pool de Conexiones TCP** del Coordinador (`make(chan *ConexionTCP, tamano)`). Esto permite que múltiples peticiones soliciten y devuelvan sockets de red de forma segura y concurrente sin generar colisiones.

---

## 2. Desarrollo de la API REST y Notificaciones en Tiempo Real

La API fue desarrollada utilizando el paquete nativo `net/http` de Go. Además de ser el servidor HTTP de exposición externa, actúa como orquestador del ciclo de vida de la petición.

### Endpoints REST implementados

Se dispone de rutas unificadas que serializan y deserializan de manera segura las matrices de *features* en formato JSON:

| Endpoint | Método | Función |
| -------- | ------ | ------- |
| `/predict/crime-type` | POST | Delegación al `nodo-model1` |
| `/predict/risk-zone` | POST | Delegación al `nodo-model2` |
| `/predict/arrest-prob` | POST | Delegación al `nodo-model3` |
| `/health` | GET | Métricas de los servidores y uso general de bases de datos |
| `/cache/stats` | GET | Evaluación de efectividad del sistema Redis (hits/misses) |

### Comunicación en Tiempo Real mediante WebSocket

Para dotar al sistema de reactividad y visibilidad en vivo, se ha integrado un **Hub de WebSocket** (`api/websocket.go`). Este componente permite mantener una conexión bidireccional, continua y persistente con los clientes web (como el panel de administrador o Dashboards Frontend).

La arquitectura emplea la librería `gorilla/websocket` y opera mediante dos mecanismos simultáneos:
1.  **Broadcast impulsado por eventos (Event-Driven)**: Tras resolverse de forma exitosa cualquier petición REST, el Handler invoca `hub.Broadcast()`. Este proceso empuja inmediatamente la nueva data a todas las instancias conectadas al socket, reportando la decisión del modelo de ML, el tiempo en milisegundos (`duracion_ms`) y el origen del cálculo.
2.  **Telemetry Heartbeat**: Un `time.Ticker` envía el "pulso vital" del sistema, transmitiendo las métricas del clúster (nodos activos y carga del sistema en la capa de coordinación) de manera automática cada 5 segundos a los administradores de la plataforma.

```go
// Fragmento simplificado del Broadcast de eventos en tiempo real
func (h *HubWS) Broadcast(msg MensajeWS) {
	data, _ := json.Marshal(msg)
	h.mu.RLock() // Bloqueo de lectura concurrente
	for c := range h.clientes {
		c.mu.Lock()
		c.conn.WriteMessage(websocket.TextMessage, data)
		c.mu.Unlock()
	}
	h.mu.RUnlock()
}
```

---

## 3. Implementación de Bases de Datos

Para maximizar el rendimiento, el soporte a grandes volúmenes de datos transaccionales, y cumplir con los patrones de arquitecturas elásticas, el sistema ha sido diseñado sobre una estructura dual de bases de datos contenerizadas.

### Caché Colaborativo en Memoria (Redis)
Las operaciones predictivas de los *Random Forests* pueden resultar intensivas, en especial bajo ataques o picos de peticiones masivas con variables predecibles. Por esto, la primera barrera de contención es **Redis**, actuando como una capa de acceso ultrarrápido (Time-to-first-byte < 1ms).

El diseño implementado en `db/redis.go` garantiza eficiencia absoluta:
1.  **Hash Criptográfico de Entrada**: Se aplica una función hash **SHA-256** al arreglo de `features` (ej. `pred:model1:ed5a1a06`). Esto crea una huella digital determinística que evita colisiones en la llave-valor.
2.  **Mecanismo de Retención Segura**: Cada predicción almacenada cuenta con un **TTL (Time to Live) de 10 minutos**, lo que previene que la memoria RAM de Redis sufra saturaciones a largo plazo y obliga a re-evaluar posibles actualizaciones en los modelos en tiempo de producción.
3.  **Degradación Elegante (Resiliencia)**: Si el motor en memoria experimenta una falla grave y cae, el servidor de Go lo detecta al inicio o durante el procesamiento, esquiva los errores silenciosamente, y todas las solicitudes fluyen transparentemente a los nodos TCP, asegurando una **alta disponibilidad**.

```go
// Generación determinística y hasheada de claves en Redis
func (r *ClienteRedis) generarClave(modelo string, features []float64) string {
	parts := make([]string, len(features))
	for i, f := range features {
		parts[i] = fmt.Sprintf("%.6f", f)
	}
	raw := fmt.Sprintf("%s:%s", modelo, strings.Join(parts, ","))
	hash := sha256.Sum256([]byte(raw)) // Hash único basado en inputs
	return fmt.Sprintf("pred:%s:%x", modelo, hash[:8])
}
```

### Almacenamiento Persistente Histórico (MongoDB)
Paralelamente a Redis, **MongoDB** funciona como el "Data Lake" del sistema. Siguiendo el principio de eficiencia de recursos y reducción de I/O de disco, el sistema persiste asíncronamente en la colección `predicciones` **únicamente los cálculos originales (Cache Miss)** resueltos por el Clúster TCP.

Si una petición es interceptada y resuelta por la caché en 0ms (Cache Hit), se considera redundante y se evita escribir en la base de datos, mitigando cuellos de botella en la persistencia durante ataques de denegación de servicio (DDoS) o picos de tráfico. Dado que esta función reside en canales no-bloqueantes, la operación de I/O de disco jamás penaliza el tiempo de respuesta.

---

## 4. Inicialización y Despliegue de la Infraestructura

Buscando mantener compatibilidad multiplataforma y uniformidad de versiones (resolviendo el problema local de *“en mi máquina sí funciona”*), la capa inferior de la plataforma de datos ha sido paquetizada mediante Docker.

El orquestador `docker-compose.yml` aprovisiona la plataforma con sus puertos nativos expuestos:
```yaml
services:
  mongodb:
    image: mongo:latest
    container_name: pc4-mongo
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db
    restart: unless-stopped

  redis:
    image: redis:latest
    container_name: pc4-redis
    ports:
      - "6379:6379"
```

El servidor monolítico principal de Go (Gateway, Coordinador, y WebSockets) se ejecuta y compila nativamente en el entorno de desarrollo, conectándose localmente tanto a los nodos TCP paralelos como a los motores en Docker, ofreciendo de esta forma la agilidad para poder depurar flujos con extremada soltura en el proceso de ingeniería de software.

---

## 5. Implementación y Pruebas Funcionales

Para asegurar la correctitud y resiliencia del sistema distribuido, se diseñó un flujo de pruebas secuencial que valida el funcionamiento de cada capa de la arquitectura implementada.

### Prueba 1: Levantamiento de Entorno e Infraestructura Dockerizada
**Objetivo:** Validar que los motores de bases de datos y los microservicios TCP arranquen sin conflictos de red.
*   **Acción:** Se ejecutó `docker compose up -d` para levantar la infraestructura de almacenamiento. Posteriormente, se ejecutó el orquestador principal en Go (`go run .`).
*   **Resultado Exitoso:** Los logs de la terminal confirmaron conexiones exitosas hacia MongoDB (`localhost:27017`) y Redis (`localhost:6379`). Simultáneamente, los 3 Nodos TCP levantaron independientemente en los puertos `:9001`, `:9002` y `:9003`, y el Coordinador reportó conexión efectiva con su *Connection Pool*.

### Prueba 2: Inferencia Distribuida y Eficiencia de Caché (Redis)
**Objetivo:** Demostrar la reducción drástica de latencia que ofrece Redis frente a cálculos matemáticos pesados repetitivos.
*   **Acción (Petición Inicial):** Se envió un *payload* JSON simulando un crimen mediante método POST a `/predict/crime-type`.
*   **Comportamiento (Cache Miss):** El sistema no encontró la huella SHA-256 en Redis. Derivó la tarea al nodo TCP `:9001`. El modelo Random Forest se ejecutó en memoria y retornó el resultado en un tiempo total de **~3 milisegundos**.
*   **Acción (Petición Repetida):** Se volvió a enviar exactamente el mismo *payload* un segundo después.
*   **Comportamiento (Cache Hit):** El sistema identificó el hash en memoria. Redis interceptó la solicitud y retornó la predicción instantáneamente en **0 milisegundos**, demostrando un ahorro del 100% en tiempo computacional del clúster TCP. Las métricas del endpoint `/cache/stats` confirmaron un "hit" exitoso.

### Prueba 3: Observabilidad en Tiempo Real (WebSockets)
**Objetivo:** Comprobar que los clientes web reciben los resultados de la analítica sin necesidad de hacer *polling* (recargar la página).
*   **Acción:** Se mantuvo abierto el cliente de pruebas en el navegador (`test_ws.html`) conectado al puerto `ws://localhost:8080/ws`. Se procedió a ejecutar peticiones POST desde la terminal.
*   **Resultado Exitoso:** Se observó que el *Telemetry Heartbeat* enviaba correctamente un pulso de salud del cluster cada 5 segundos. Además, cada vez que una predicción era generada en la terminal, la interfaz web recibía un evento de broadcast instantáneo mostrando el modelo usado, el resultado, y si la data provino del caché o del servidor TCP.

### Prueba 4: Persistencia Asíncrona (MongoDB)
**Objetivo:** Validar que el Data Lake histórico registre de forma no bloqueante las predicciones únicas (Cache Miss), ahorrando I/O en peticiones redundantes.
*   **Acción:** Tras realizar 5 predicciones (3 nuevas y 2 cacheadas), se consultó la colección `predicciones` utilizando MongoDB Compass y el endpoint `GET /predictions`.
*   **Resultado Exitoso:** La base de datos persistió correctamente únicamente los **3 documentos originales** con sus respectivos identificadores `ObjectID`, demostrando el correcto filtrado de la caché para ahorrar I/O de disco. Los vectores de características (features) y los tiempos de procesamiento se guardaron fielmente sin bloquear las respuestas del usuario final.
