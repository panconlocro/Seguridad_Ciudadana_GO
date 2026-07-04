1ACC0065 - Programación Concurrente y Distribuida
Informe de Trabajo Final — Entregable 3 (TB2)
Carrera de Ciencias de la Computación
Docente: Carlos Alberto Jara García
Nombres y Apellidos Código
Rosa María Rodríguez Valencia U202212675
Mildred Micaela Marchan Quispe U202213292
Nicolas Francisco Miranda Rafael U202216895
2026 - 1

---

## 1. Introducción y Resumen de Entregas Previas

Este documento constituye el informe del **Entregable 3 (TB2)**, la consolidación final del proyecto **SecurityGO**: una plataforma de predicción de seguridad ciudadana basada en Machine Learning distribuido, programada íntegramente en Go (backend) y React (frontend).

### Entregas anteriores integradas

- **PC3 (Entregable 1):** Entrenamiento de 3 modelos Random Forest con Python (scikit-learn), exportados a JSON. Concurrencia básica con goroutines y channels para inferencia.
- **PC4 (Entregable 2):** Arquitectura distribuida con Cluster TCP (Coordinador + Nodos Worker), API REST con `net/http`, WebSocket para eventos en tiempo real, Redis como caché (SHA-256 + TTL), MongoDB como Data Lake asíncrono, y Docker Compose para la infraestructura.

---

## 2. Desarrollo del Frontend (SPA)

### Stack tecnológico

| Componente | Tecnología | Justificación |
| ---------- | ---------- | ------------- |
| Framework | React (Vite) | Build ultra-rápido, HMR instantáneo |
| Estilos | CSS Vanilla (Design Tokens) | Tema "Vigía Nocturna" (Amber/Teal), Grid background |
| Gráficos | Recharts | Integración nativa con React, SVG responsive |
| Estado global | React Context API | Manejo de sesión JWT sin dependencias extra |

### Diseño e Interfaz Gráfica ("Vigía Nocturna")

Para el entregable final se rediseñó completamente la interfaz bajo el concepto **"Vigía Nocturna"**, emulando un centro de operaciones urbano de seguridad. Se implementó una identidad visual propia con:
- **Paleta de colores:** Fondo oscuro (`--midnight`), acentos en ámbar (`--signal-amber`) para acciones y verde azulado (`--grid-teal`) para datos positivos.
- **Tipografía:** *Space Grotesk* para títulos (display), *DM Sans* para el cuerpo y *JetBrains Mono* para datos técnicos.
- **Layout:** *Top navigation bar* horizontal (reemplazando la antigua sidebar) para maximizar el espacio de formularios y gráficos.
- **Detalles Visuales:** Animación sutil de *scan-line* y fondo de grilla urbana (radar), abandonando el glassmorphism genérico.

### Componentes implementados

#### 2.1 Sistema de Autenticación (Login)

El componente `Login.jsx` presenta una interfaz de autenticación moderna, con el logo CSS integrado de SecurityGO y un layout limpio (con link toggle). Tras ingresar credenciales válidas, se realiza una petición `POST /login` al backend Go, que retorna un **token JWT firmado con HS256**. Este token se almacena en `localStorage` y se adjunta automáticamente en todas las peticiones subsiguientes mediante el header `Authorization: Bearer <token>`.

El `AuthContext.jsx` gestiona el estado global de sesión (usuario, rol, token) y expone funciones `login()` y `logout()` para todos los componentes hijos.

#### 2.2 Panel de Predicciones

El componente `Predicciones.jsx` implementa formularios dinámicos para los 3 modelos de ML:

1. **Clasificación de Tipo de Crimen** (model1 → `:9001`)
2. **Predicción de Zona de Riesgo** (model2 → `:9002`)
3. **Estimación de Probabilidad de Arresto** (model3 → `:9003`)

Cada formulario se genera automáticamente a partir de la configuración del modelo, incluyendo validaciones de rango. Los resultados muestran:
- La predicción del modelo
- Indicador de origen (Cache Hit ⚡ vs TCP Cluster 🖥️)
- Tiempo de respuesta en milisegundos
- Nodo worker que procesó la solicitud
- Vista JSON expandible del payload completo

#### 2.3 Panel de Administración (Tiempo Real)

El componente `AdminPanel.jsx` se conecta al endpoint WebSocket (`ws://localhost:8080/ws`) para recibir:

- **Eventos de predicción:** Cada vez que cualquier cliente ejecuta una predicción, el evento se propaga instantáneamente vía broadcast.
- **Métricas del cluster:** Cada 5 segundos, el Telemetry Heartbeat envía el estado de los nodos TCP (activos, predicciones acumuladas).

Visualizaciones con Recharts:
- **Gráfico de barras:** Predicciones acumuladas por nodo TCP (adaptado a la nueva paleta amber/teal)
- **Gráfico de pastel:** Ratio de Cache Hit/Miss de Redis
- **Stream de eventos:** Lista en tiempo real con indicadores de cache (amber/teal) y latencia monospace

Los KPIs se presentan en un *stat strip* horizontal unificado: nodos activos, predicciones totales, registros en MongoDB, y clientes WebSocket conectados.

#### 2.4 Historial y Análisis de Impacto

El componente `Historial.jsx` consulta el endpoint `GET /predictions` (alimentado por MongoDB) y presenta:

- **Distribución por modelo:** Gráfico de barras con la cantidad de predicciones por modelo
- **Timeline de actividad:** Gráfico de líneas (amber) mostrando predicciones agrupadas por hora
- **Latencia promedio:** *Stat strip* con métricas numéricas de rendimiento por modelo
- **Tabla de registros:** Filas compactas con timestamp, modelo, nodo, duración y resultado (usando monospace para datos técnicos)

Se incluye una barra horizontal de filtros (*inline*) por modelo y límite de resultados.

---

## 3. Autenticación JWT en el Backend Go

### Implementación (`api/auth.go`)

Se añadió un módulo de autenticación basado en **JSON Web Tokens (JWT)** utilizando la librería `github.com/golang-jwt/jwt/v5`.

**Flujo de autenticación:**

1. El usuario envía `POST /login` con credenciales JSON `{usuario, password}`
2. El servidor valida contra una tabla de usuarios estáticos (extensible a MongoDB)
3. Si las credenciales son válidas, se genera un token JWT firmado con HMAC-SHA256 que contiene:
   - `usuario`: nombre del usuario autenticado
   - `rol`: nivel de acceso ("admin" o "user")
   - `exp`: expiración a las 24 horas
   - `iat`: timestamp de emisión
   - `iss`: "SecurityGO-TB2"
4. El token se retorna al cliente para uso en requests subsiguientes

**Middleware JWT:**

Las rutas de predicción (`/predict/*`), historial (`/predictions`) y caché (`/cache/*`) están protegidas por el middleware `JWTMiddleware`, que:
1. Extrae el header `Authorization: Bearer <token>`
2. Parsea y valida la firma HMAC-SHA256
3. Verifica la expiración del token
4. Si es válido, permite el paso al handler; si no, retorna `401 Unauthorized`

Las rutas públicas (`/login`, `/health`, `/ws`) permanecen sin protección.

---

## 4. Arquitectura Final Integrada

```
┌─────────────────────────────────────────────────────────────────┐
│                     FRONTEND (React + Vite)                     │
│  ┌──────────┐  ┌─────────────┐  ┌──────────┐  ┌─────────────┐  │
│  │  Login   │  │ Predicciones│  │  Admin   │  │  Historial  │  │
│  │  (JWT)   │  │ (3 modelos) │  │  (WS)    │  │  (Gráficos) │  │
│  └────┬─────┘  └──────┬──────┘  └────┬─────┘  └──────┬──────┘  │
│       │               │              │               │          │
│       └───────────────┼──────────────┼───────────────┘          │
│                       │     HTTP + JWT / WebSocket               │
└───────────────────────┼──────────────┼──────────────────────────┘
                        │              │
┌───────────────────────┼──────────────┼──────────────────────────┐
│                  API GATEWAY (Go :8080)                          │
│   ┌───────────┐  ┌───────────┐  ┌─────────┐  ┌─────────────┐   │
│   │ JWT Auth  │  │ Handlers  │  │   CORS  │  │  WS Hub     │   │
│   │ Middleware│  │  REST     │  │ Middleware│ │  (Broadcast) │   │
│   └───────────┘  └─────┬─────┘  └─────────┘  └──────┬──────┘   │
│                        │                             │          │
│              ┌─────────┴─────────┐                   │          │
│              │  COORDINADOR TCP  │◄──────────────────┘          │
│              │  (Connection Pool)│                               │
│              └──┬───────┬───────┬┘                              │
│                 │       │       │        TCP (JSON\n)            │
└─────────────────┼───────┼───────┼───────────────────────────────┘
                  │       │       │
    ┌─────────────┤       │       ├─────────────┐
    │             │       │       │             │
┌───┴───┐   ┌────┴───┐  ┌┴──────┐             │
│Nodo ML│   │Nodo ML │  │Nodo ML│             │
│:9001  │   │:9002   │  │:9003  │             │
│model1 │   │model2  │  │model3 │             │
└───────┘   └────────┘  └───────┘             │
                                              │
┌─────────────────────────────────────────────┤
│          INFRAESTRUCTURA (Docker)           │
│  ┌──────────────┐  ┌───────────────────┐    │
│  │   MongoDB    │  │      Redis        │    │
│  │   :27017     │  │     :6379         │    │
│  │  (Data Lake) │  │  (Cache SHA-256)  │    │
│  └──────────────┘  └───────────────────┘    │
└─────────────────────────────────────────────┘
```

---

## 5. Mecanismos de Concurrencia y Distribución (Consolidación)

### Concurrencia (Go)
| Mecanismo | Ubicación | Propósito |
| --------- | --------- | --------- |
| **Goroutines** | `tcp_node.go`, `handlers.go` | Cada conexión TCP y cada escritura a MongoDB/Redis se ejecuta en goroutines independientes |
| **Channels** | `coordinator.go` | Pool de conexiones TCP implementado como canal buffered (`chan *ConexionTCP`) |
| **sync/atomic** | `coordinator.go`, `redis.go` | Contadores de predicciones y estadísticas de cache sin mutex |
| **sync.RWMutex** | `websocket.go` | Protección de lectura/escritura del mapa de clientes WebSocket |

### Distribución
| Aspecto | Implementación |
| ------- | -------------- |
| **Comunicación** | Protocolo TCP con mensajes JSON delimitados por `\n` |
| **Topología** | Coordinador central → 3 nodos worker independientes |
| **Resiliencia** | Redis en modo degradado, connection pooling con reconexión |
| **Persistencia** | MongoDB asíncrono (solo cache miss) |
| **Tiempo real** | WebSocket broadcast con heartbeat de 5 segundos |

---

## 6. Pruebas Funcionales (Entregable 3)

### Prueba 5: Autenticación JWT End-to-End
**Objetivo:** Validar que el flujo de login → token → acceso protegido funciona correctamente.
- **Acción:** Se ingresa al frontend, se proporciona usuario `admin` / contraseña `admin123`.
- **Resultado:** El servidor retorna un token JWT válido. Las peticiones de predicción se envían con header `Authorization: Bearer <token>`. Si se intenta acceder sin token, el servidor retorna `401 Unauthorized`.

### Prueba 6: Interacción Frontend ↔ Cluster TCP
**Objetivo:** Comprobar que el formulario de predicción del frontend ejecuta correctamente el flujo distribuido completo.
- **Acción:** Desde el panel de predicciones, se completa el formulario de "Tipo de Crimen" y se presiona "Ejecutar Predicción".
- **Resultado:** La UI muestra el resultado con indicador de origen (Cache/TCP), latencia, y nodo worker. Simultáneamente, el Panel Admin recibe el evento via WebSocket en tiempo real.

### Prueba 7: Visualización de Métricas en Tiempo Real
**Objetivo:** Demostrar la reactividad del sistema con múltiples clientes conectados.
- **Acción:** Se abren dos pestañas del navegador: una en el panel de predicciones y otra en el panel admin.
- **Resultado:** Cada predicción ejecutada desde la primera pestaña se refleja instantáneamente en el stream de eventos de la segunda pestaña, con gráficos actualizados dinámicamente.

---

## 7. Instrucciones de Ejecución

```bash
# 1. Levantar infraestructura de bases de datos
cd PC4
docker compose up -d

# 2. Iniciar el backend Go (API + Cluster TCP)
cd PC4
go run .

# 3. Iniciar el frontend en modo desarrollo
cd frontend
npm install
npm run dev
```

**Credenciales de prueba:**
| Usuario | Contraseña | Rol |
| ------- | ---------- | --- |
| admin | admin123 | admin |
| user | user123 | user |
| rosa | seguridad2026 | admin |

---

## 8. Conclusiones

El proyecto SecurityGO demuestra una integración completa de los conceptos de **programación concurrente y distribuida** aplicados a un problema real de seguridad ciudadana. El sistema final combina:

1. **Concurrencia nativa de Go** (goroutines, channels, atomic operations) para maximizar el throughput bajo carga.
2. **Distribución real con TCP** mediante un cluster de nodos ML independientes con connection pooling.
3. **Persistencia dual** (Redis + MongoDB) con estrategias inteligentes de caché y escritura asíncrona.
4. **Comunicación en tiempo real** via WebSocket para observabilidad del sistema.
5. **Interfaz web moderna** con React, autenticación JWT, y visualizaciones interactivas.

El diseño modular permite escalar horizontalmente agregando más nodos TCP sin modificar el frontend ni el coordinador, demostrando los principios de arquitectura distribuida estudiados en el curso.
