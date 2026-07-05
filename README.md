# Seguridad Ciudadana GO

Este proyecto es un sistema de análisis, modelado predictivo y visualización (con backend en Go y frontend) enfocado en seguridad ciudadana utilizando datos criminales.

## Guía de Reproducibilidad

Sigue estos pasos estrictamente en orden para preparar el entorno, procesar los datos, entrenar los modelos y levantar toda la aplicación.

### 1. Entorno Virtual (Jupyter / Python)
Si vas a utilizar los notebooks de exploración y experimentación, configura el entorno virtual y las dependencias (Jupyter):

**En Windows:**
```powershell
.\setup_venv.ps1
```

**En Linux / macOS:**
```bash
./setup_venv.sh
```

### 2. Descarga de Datos (Download Data)
Consigue el dataset original y ubícalo en la carpeta cruda (raw). Asegúrate de que la ruta final quede así:
`data/raw/Crime_Data_from_2020_to_Present.csv`

### 3. Limpieza de Datos (Cleanse)
Procesa el dataset original para limpiarlo y generar las variables (features) que utilizarán los modelos:

```powershell
go run scripts/Cleanse/main.go
```
*Este proceso generará el archivo limpio en `data/processed/Crime_Data_Clean.csv`.*

### 4. Entrenamiento de Modelos (Models)
Entrena y evalúa los tres modelos predictivos. Ejecuta los siguientes comandos uno por uno desde la raíz del proyecto:

```powershell
# Modelo 1: Clasificación de tipo de crimen
.\run_models.ps1 run --model-type model1

# Modelo 2: Predicción de zona de riesgo (coordenadas)
.\run_models.ps1 run --model-type model2

# Modelo 3: Probabilidad de arresto
.\run_models.ps1 run --model-type model3
```
*Los modelos se entrenarán y se guardarán como archivos `.json` dentro de la carpeta `models/`.*

### 5. Servicios de Base de Datos (Docker)
El backend requiere MongoDB y Redis para manejar sesiones y almacenar datos. Utiliza Docker Compose para levantarlos:

```powershell
docker-compose up -d
```

### 6. Backend (Go API & TCP Cluster)
Una vez que los modelos estén generados y las bases de datos corriendo, inicia el servidor backend. Desde la raíz, en una terminal nueva:

```powershell
cd backend
go run .
```
*El servidor verificará la conexión a Mongo/Redis y levantará los nodos TCP con sus respectivos modelos JSON.*

### 7. Frontend
Levanta la interfaz web de la aplicación. Desde la raíz, en una terminal nueva:

```powershell
cd frontend
npm install
npm run dev
```
*Ingresa al enlace local (usualmente `http://localhost:5173`) provisto en la consola para usar la aplicación.*

---

## TODO
- [ ] Agregar pruebas automatizadas (tests) para el backend y los modelos.
- [ ] Optimización de hiperparámetros de los Random Forest para mejorar precisión.
- [ ] Implementar un pipeline de CI/CD para automatizar tests y linting.
- [ ] Refinar la interfaz y experiencia de usuario (UI/UX) en el Frontend.
