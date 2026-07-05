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
Consigue el dataset original y ubícalo en la carpeta cruda (raw). 

Desde el repositorio: 

`py scripts\download_dataset.py`

Asegúrate de que la ruta final quede así:

`data/raw/Crime_Data_from_2020_to_Present.csv`

### 3. Limpieza de Datos (Cleanse)
Procesa el dataset original para limpiarlo y generar las variables (features) que utilizarán los modelos:

```powershell
go run scripts/Cleanse/main.go
```
*Este proceso generará el archivo limpio en `data/processed/Crime_Data_Clean.csv`.*

### 4. Entrenamiento de Modelos (Backend API)
El entrenamiento y persistencia de los modelos ahora se gestionan dinámicamente mediante el **backend** de la aplicación utilizando la API (`POST /train`) y **MongoDB GridFS**. Ya no es necesario ejecutar scripts en la terminal para generar archivos locales; basta con tener las bases de datos y el servidor backend corriendo.

### 5. Servicios de Base de Datos (Docker)
El backend requiere MongoDB (incluyendo soporte de GridFS) y Redis para manejar sesiones y almacenar datos. Utiliza Docker Compose para levantarlos:

```powershell
docker-compose up -d
```

### 6. Backend (Go API & TCP Cluster)
Una vez que las bases de datos estén corriendo, inicia el servidor backend. Desde la raíz, en una terminal nueva:

```powershell
cd backend
.\run_backend.ps1
```
*(También puedes ejecutar `go run .` directamente).*
*El servidor verificará la conexión a Mongo/Redis, descargará los últimos modelos desde GridFS (si existen), y levantará los nodos TCP locales listos para servir o recibir nuevos entrenamientos.*

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
