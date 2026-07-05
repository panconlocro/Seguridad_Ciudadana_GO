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
- **Detalles Visuales:** Animación ambiental de *radar-sweep* en el fondo de grilla urbana (optimizada para GPU mediante `will-change: transform` y respetando accesibilidad con `prefers-reduced-motion`), abandonando el glassmorphism genérico.

### Componentes implementados

#### 2.1 Sistema de Autenticación y Registro (Login + Register)

El componente `Login.jsx` presenta una interfaz de autenticación moderna con **dos pestañas**: "Iniciar Sesión" y "Crear Cuenta". Los usuarios nuevos pueden registrarse directamente desde la interfaz, enviando un `POST /register` al backend, que cifra la contraseña con **bcrypt** y la almacena en MongoDB. Para el inicio de sesión, se realiza una petición `POST /login` que valida las credenciales contra la base de datos y retorna un **token JWT firmado con HS256**. Este token se almacena en `localStorage` y se adjunta automáticamente en todas las peticiones subsiguientes mediante el header `Authorization: Bearer <token>`.

Validaciones implementadas en el formulario de registro:
- Nombre de usuario: mínimo 3 caracteres
- Contraseña: mínimo 6 caracteres
- Detección de usuario duplicado (error 409 desde el backend)
- Mensaje de éxito con transición automática a la pestaña de login

El `AuthContext.jsx` gestiona el estado global de sesión (usuario, rol, token) y expone funciones `login()` y `logout()` para todos los componentes hijos.

#### 2.2 Panel de Predicciones

El componente `Predicciones.jsx` implementa formularios dinámicos para los 3 modelos de ML:

1. **Clasificación de Tipo de Crimen** (model1 → `:9001`)
2. **Predicción de Zona de Riesgo** (model2 → `:9002`)
3. **Estimación de Probabilidad de Arresto** (model3 → `:9003`)

Cada formulario se genera automáticamente a partir de la configuración del modelo, incluyendo validaciones de rango y **tooltips informativos** (globos de ayuda visuales) para explicar cada variable al usuario. Los resultados se renderizan con un **diseño amigable y adaptado al tipo de modelo** (ej. colores dinámicos de severidad o enlaces directos a Google Maps), e incluyen:
- La predicción final del modelo
- Indicador de origen (Cache Hit ⚡ vs TCP Cluster 🖥️)
- Tiempo de respuesta en milisegundos
- Nodo worker que procesó la solicitud
- Vista JSON expandible del payload completo para auditoría

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

## 3. Autenticación y Registro en el Backend Go

### Implementación (`api/auth.go` + `db/mongo.go`)

Se implementó un sistema completo de autenticación y registro de usuarios basado en **JSON Web Tokens (JWT)** con `github.com/golang-jwt/jwt/v5` y cifrado de contraseñas con **bcrypt** (`golang.org/x/crypto/bcrypt`).

**Almacenamiento de usuarios (MongoDB):**

Los usuarios se almacenan en la colección `usuarios` de MongoDB con la siguiente estructura:
- `usuario`: nombre único del usuario
- `password_hash`: contraseña cifrada con bcrypt (costo default = 10)
- `rol`: nivel de acceso (`"admin"` o `"user"`)
- `creado_en`: timestamp UTC de creación

Al iniciar el servidor, la función `InicializarAdmin()` verifica si existe al menos un usuario con rol `admin`. Si no lo hay, crea automáticamente la cuenta `admin` / `admin123` con contraseña cifrada.

**Flujo de registro (`POST /register`):**

1. El usuario envía `{usuario, password}` en JSON
2. Se validan longitudes mínimas (usuario ≥ 3, password ≥ 6)
3. Se verifica que el nombre de usuario no esté duplicado en MongoDB
4. Se cifra la contraseña con `bcrypt.GenerateFromPassword`
5. Se inserta el nuevo usuario con rol `"user"` por defecto
6. Se retorna confirmación de creación exitosa

**Flujo de autenticación (`POST /login`):**

1. El usuario envía `{usuario, password}` en JSON
2. Se busca el usuario en la colección `usuarios` de MongoDB
3. Se compara la contraseña con el hash almacenado usando `bcrypt.CompareHashAndPassword`
4. Si las credenciales son válidas, se genera un token JWT firmado con HMAC-SHA256 que contiene:
   - `usuario`: nombre del usuario autenticado
   - `rol`: nivel de acceso ("admin" o "user")
   - `exp`: expiración a las 24 horas
   - `iat`: timestamp de emisión
   - `iss`: "SecurityGO-TB2"
5. El token se retorna al cliente para uso en requests subsiguientes

**Middleware JWT:**

Las rutas de predicción (`/predict/*`), historial (`/predictions`) y caché (`/cache/*`) están protegidas por el middleware `JWTMiddleware`, que:
1. Extrae el header `Authorization: Bearer <token>`
2. Parsea y valida la firma HMAC-SHA256
3. Verifica la expiración del token
4. Si es válido, permite el paso al handler; si no, retorna `401 Unauthorized`

Las rutas públicas (`/login`, `/register`, `/health`, `/ws`) permanecen sin protección.

---

## 4. Arquitectura Final Integrada

### Estructura del Monorepo

El proyecto sigue una estructura monorepo organizada por responsabilidad:

```
Seguridad_Ciudadana_GO/
├── docker-compose.yml          # Infraestructura (MongoDB + Redis)
├── backend/                    # Servidor Go (API + Cluster TCP)
│   ├── main.go
│   ├── api/                    # Handlers REST, Auth JWT, WebSocket
│   │   ├── auth.go             # Login + Register + JWT Middleware
│   │   ├── handlers.go         # Endpoints de predicción
│   │   ├── server.go           # Configuración y rutas
│   │   └── websocket.go        # Hub WebSocket
│   ├── cluster/                # Cluster TCP distribuido
│   │   ├── coordinator.go      # Coordinador con connection pool
│   │   ├── tcp_node.go         # Nodos TCP (listeners)
│   │   ├── worker.go           # Inferencia ML (Random Forest)
│   │   └── tipos.go            # Tipos compartidos
│   └── db/                     # Capa de datos
│       ├── mongo.go            # MongoDB (predicciones + usuarios)
│       └── redis.go            # Redis (caché SHA-256)
├── frontend/                   # SPA React + Vite
│   └── src/
│       ├── components/         # Login, Predicciones, AdminPanel, Historial
│       ├── context/            # AuthContext (sesión JWT)
│       └── api.js              # Cliente HTTP centralizado
├── models/                     # Modelos ML exportados (JSON)
└── docs/                       # Documentación
```

### Diagrama de Arquitectura

```
┌─────────────────────────────────────────────────────────────────┐
│                     FRONTEND (React + Vite)                     │
│  ┌──────────┐  ┌─────────────┐  ┌──────────┐  ┌─────────────┐  │
│  │  Login/  │  │ Predicciones│  │  Admin   │  │  Historial  │  │
│  │ Register │  │ (3 modelos) │  │  (WS)    │  │  (Gráficos) │  │
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
│   │ + bcrypt  │  │  REST     │  │ Middleware│ │  (Broadcast) │   │
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
│  │  (Data Lake  │  │  (Cache SHA-256)  │    │
│  │  + Usuarios) │  │                   │    │
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

### Prueba 5: Registro de Usuario y Autenticación JWT End-to-End
**Objetivo:** Validar el flujo completo de registro → login → token → acceso protegido.
- **Acción:** Se ingresa al frontend, se selecciona la pestaña "Crear Cuenta", se registra un nuevo usuario con contraseña. Luego se inicia sesión con las credenciales recién creadas.
- **Resultado:** El registro cifra la contraseña con bcrypt y la almacena en MongoDB. El login posterior valida el hash y retorna un token JWT válido. Las peticiones de predicción se envían con header `Authorization: Bearer <token>`. Si se intenta acceder sin token, el servidor retorna `401 Unauthorized`. Si se intenta registrar un usuario duplicado, retorna `409 Conflict`.

### Prueba 6: Interacción Frontend ↔ Cluster TCP
**Objetivo:** Comprobar que el formulario de predicción del frontend ejecuta correctamente el flujo distribuido completo.
- **Acción:** Desde el panel de predicciones, se completa el formulario de "Tipo de Crimen" y se presiona "Ejecutar Predicción".
- **Resultado:** La UI muestra el resultado con indicador de origen (Cache/TCP), latencia, y nodo worker. Simultáneamente, el Panel Admin recibe el evento via WebSocket en tiempo real.

### Prueba 7: Visualización de Métricas en Tiempo Real
**Objetivo:** Demostrar la reactividad del sistema con múltiples clientes conectados.
- **Acción:** Se abren dos pestañas del navegador: una en el panel de predicciones y otra en el panel admin.
- **Resultado:** Cada predicción ejecutada desde la primera pestaña se refleja instantáneamente en el stream de eventos de la segunda pestaña, con gráficos actualizados dinámicamente.

### Prueba 8: Historial de Predicciones con MongoDB
**Objetivo:** Verificar que las predicciones se persisten correctamente y se visualizan con gráficos estadísticos.
- **Acción:** Se ejecutan múltiples predicciones con distintos modelos. Se navega a la pestaña "Historial".
- **Resultado:** Los gráficos de distribución por modelo, actividad en el tiempo y latencia promedio se pueblan con los datos reales almacenados en MongoDB. La tabla de registros muestra timestamp, modelo, nodo, duración y resultado de cada predicción. Los filtros por modelo y límite funcionan correctamente.

---

## 7. Instrucciones de Ejecución

```bash
# 1. Levantar infraestructura de bases de datos
docker compose up -d

# 2. Iniciar el backend Go (API + Cluster TCP)
cd backend
go run .

# 3. Iniciar el frontend en modo desarrollo
cd frontend
npm install
npm run dev
```

**Cuenta administrador por defecto:**

Al iniciar el backend por primera vez, se crea automáticamente un usuario administrador si no existe ninguno en la base de datos:

| Usuario | Contraseña | Rol | Creación |
| ------- | ---------- | --- | -------- |
| admin | admin123 | admin | Automática (si no hay admins) |

Los demás usuarios se registran desde la interfaz web (pestaña "Crear Cuenta") con rol `user` por defecto.

**Endpoints disponibles:**

| Método | Ruta | Protegido | Descripción |
| ------ | ---- | --------- | ----------- |
| POST | `/login` | No | Autenticación con JWT |
| POST | `/register` | No | Registro de nuevo usuario |
| GET | `/health` | No | Estado del sistema |
| WS | `/ws` | No | WebSocket (eventos en tiempo real) |
| POST | `/predict/crime-type` | JWT | Predicción tipo de crimen |
| POST | `/predict/risk-zone` | JWT | Predicción zona de riesgo |
| POST | `/predict/arrest-prob` | JWT | Predicción prob. de arresto |
| GET | `/predictions` | JWT | Historial de predicciones |
| GET | `/cache/stats` | JWT | Estadísticas de caché Redis |


---

## 8. Conclusiones

El proyecto SecurityGO demuestra una integración completa de los conceptos de **programación concurrente y distribuida** aplicados a un problema real de seguridad ciudadana. El sistema final combina:

1. **Concurrencia nativa de Go** (goroutines, channels, atomic operations) para maximizar el throughput bajo carga.
2. **Distribución real con TCP** mediante un cluster de nodos ML independientes con connection pooling.
3. **Persistencia dual** (Redis + MongoDB) con estrategias inteligentes de caché y escritura asíncrona.
4. **Comunicación en tiempo real** via WebSocket para observabilidad del sistema.
5. **Interfaz web moderna** con React, autenticación JWT, y visualizaciones interactivas.
6. **Seguridad** con cifrado bcrypt para contraseñas, tokens JWT con expiración, y middleware de autorización en todas las rutas sensibles.

El diseño modular permite escalar horizontalmente agregando más nodos TCP sin modificar el frontend ni el coordinador, demostrando los principios de arquitectura distribuida estudiados en el curso.
