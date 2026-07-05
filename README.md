# Seguridad Ciudadana GO

Este proyecto es un sistema de análisis, modelado predictivo y visualización (con backend en Go y frontend en React/Vite) enfocado en seguridad ciudadana utilizando datos criminales reales.

## 🌐 Pruebas en Vivo (Live Demo)
Puedes probar el proyecto ya desplegado y funcionando en la nube en el siguiente enlace:
- **[https://tu-proyecto.netlify.app](https://tu-proyecto.netlify.app)** *(Reemplaza con tu URL real de Netlify)*

---

## 🚀 Cómo correr el proyecto localmente

Si quieres levantar el proyecto en tu propia máquina para probar el Frontend y el Backend conectados a tus bases de datos, sigue estos pasos:

### 1. Configurar Bases de Datos
Necesitas tener un clúster de **MongoDB** y uno de **Redis**. Puedes usar servicios en la nube (como Mongo Atlas y Upstash) o levantarlos localmente con Docker:
```powershell
docker-compose up -d
```

### 2. Subir Modelos a MongoDB (Solo la primera vez)
El backend requiere que los modelos predictivos estén en la base de datos. Si tu base de datos está vacía, sube los modelos ejecutando esto desde una terminal:
```powershell
cd backend
go run . --mongo "<TU_MONGO_URI>" --upload-models
```
*(Si usas docker local, el URI es `mongodb://localhost:27017`)*

### 3. Iniciar el Backend
Inicia el servidor backend pasándole tus credenciales de conexión:
```powershell
cd backend
go run . --mongo "<TU_MONGO_URI>" --redis "<TU_REDIS_URI>"
```
*(Si usas docker local: `go run . --mongo "mongodb://localhost:27017" --redis "redis://localhost:6379"`).*
*El servidor verificará la conexión a Mongo/Redis y levantará los nodos TCP descargando los modelos dinámicamente.*

### 4. Iniciar el Frontend
En otra terminal, levanta la interfaz web:
```powershell
cd frontend
npm install
npm run dev
```
Entra a `http://localhost:5173` para usar la aplicación.

---

## 🔬 Experimentación y Entrenamiento (Opcional)
Si deseas explorar los datos originales, limpiarlos o reentrenar los modelos de Machine Learning desde cero (solo para propósitos de ciencia de datos):

1. **Entorno Virtual**: Ejecuta `.\setup_venv.ps1` (Windows) o `./setup_venv.sh` (Linux).
2. **Descarga**: Consigue los datos crudos y ubícalos en `data/raw/Crime_Data_from_2020_to_Present.csv`.
3. **Limpieza**: Ejecuta `go run scripts/Cleanse/main.go` para generar el dataset limpio.
4. **Entrenamiento**: Entrena los modelos ejecutando `.\run_models.ps1 run --model-type model1` (repite para `model2` y `model3`). Esto sobrescribirá los archivos `.json` en la carpeta `models/`. Luego tendrás que volver a subirlos a MongoDB (Paso 2).

---

## TODO
- [ ] Agregar pruebas automatizadas (tests) para el backend y los modelos.
- [ ] Optimización de hiperparámetros de los Random Forest para mejorar precisión.
- [ ] Implementar un pipeline de CI/CD para automatizar tests y linting.
- [ ] Refinar la interfaz y experiencia de usuario (UI/UX) en el Frontend.
