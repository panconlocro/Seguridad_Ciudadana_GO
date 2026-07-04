# PC4 — SecurityGO (Cluster TCP + Redis + WebSocket)

Este directorio contiene la implementación del **Entregable 2 (PC4)**. El sistema ha evolucionado de una API monolítica simple a una **arquitectura distribuida** con comunicación por red, caché en memoria y notificaciones en vivo.

Para ver los detalles técnicos de la arquitectura y decisiones de diseño, consulta: [docs/PC4_documentacion.md](../docs/PC4_documentacion.md).

---

## 🚀 Requisitos Previos

1. **Modelos JSON**: El sistema necesita los modelos de Machine Learning pre-entrenados en la carpeta `models/` del directorio raíz.
   > *Nota: Si no los tienes, ejecuta el script `run_models.ps1 train ...` desde la raíz.*
2. **Docker Compose**: Se utiliza para levantar rápidamente la infraestructura (MongoDB y Redis).

---

## 🛠️ Cómo Ejecutar el Sistema

### Paso 1: Levantar Bases de Datos (Docker)
En una terminal, ubícate en la carpeta `PC4` y ejecuta:
```powershell
docker compose up -d
```
Esto levantará **MongoDB** en el puerto `27017` y **Redis** en el puerto `6379`.

### Paso 2: Ejecutar el Servidor Go (API + Cluster)
En la misma carpeta `PC4`, instala las dependencias e inicia el cluster:
```powershell
go mod tidy
go run .
```
El servidor iniciará:
1. 3 Nodos TCP (Workers) en los puertos `:9001`, `:9002` y `:9003`.
2. El Coordinador TCP que gestiona el Connection Pool hacia los nodos.
3. El Gateway API escuchando en `http://localhost:8080`.

### Paso 3: Monitor de Tiempo Real
Abre el archivo **`test_ws.html`** (haciendo doble clic) en tu navegador. Esto te conectará al servidor WebSocket para ver métricas y predicciones en vivo.

---

## 📡 Endpoints Disponibles

### API REST (HTTP)

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| **POST** | `/predict/crime-type` | Predice el tipo de crimen (Modelo 1). Verifica caché en Redis; si no existe, consulta al nodo TCP. |
| **POST** | `/predict/risk-zone` | Predice lat/lon de riesgo (Modelo 2). |
| **POST** | `/predict/arrest-prob` | Predice probabilidad de arresto (Modelo 3). |
| **GET** | `/health` | Estado del cluster TCP, contadores de MongoDB y Redis. |
| **GET** | `/predictions` | Historial guardado en MongoDB (acepta `?model=model1&limit=5`). |
| **GET** | `/cache/stats` | Estadísticas en vivo de Redis (hits, misses, total keys). |

### Tiempo Real (WebSocket)

| URL | Descripción |
|-----|-------------|
| `ws://localhost:8080/ws` | Emite métricas del cluster cada 5s y envía un evento inmediato cada vez que se resuelve una predicción. |

---

## 🧪 Comandos Rápidos de Prueba (PowerShell)

**1. Probar Predicción (La primera vez tardará ~3ms, la segunda vez tardará 0ms gracias a Redis):**
```powershell
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/predict/crime-type" -Headers @{"Content-Type"="application/json"} -Body '{"hour": 15, "day_of_week": 2, "month": 6, "area": 1, "premis_cd": 101, "part_1_2": 1, "victim_identified": 1, "days_to_report": 0}'
```

**2. Verificar Estadísticas del Caché Redis:**
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/cache/stats"
```

**3. Verificar Estado General del Cluster TCP:**
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/health"
```
