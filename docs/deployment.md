# Documentación de Despliegue y Arquitectura en la Nube

Este documento registra los cambios arquitectónicos y las configuraciones técnicas implementadas para migrar el sistema "Seguridad Ciudadana GO" de un entorno local a una arquitectura distribuida en la nube.

## Arquitectura del Sistema Distribuido

La solución se dividió en cuatro componentes principales alojados en servicios en la nube independientes:

1.  **Frontend (Netlify):** 
    Aplicación SPA construida con React y Vite. Se configuró para ser servida como archivos estáticos a través de un CDN, consumiendo la API del backend mediante variables de entorno dinámicas.
2.  **Backend (Render):** 
    Servidor HTTP y clúster de nodos TCP escrito en Go. Se ejecuta dentro de un contenedor Docker aislado y gestiona la lógica de Machine Learning, la persistencia y las comunicaciones en tiempo real (WebSockets).
3.  **Base de Datos Principal (MongoDB Atlas):** 
    Servicio de base de datos NoSQL alojado en la nube. Almacena las colecciones de usuarios, el historial de predicciones y, de forma crucial, los grafos de los modelos de árboles de decisión en formato JSON.
4.  **Caché (Upstash Redis):** 
    Instancia Redis *Serverless* utilizada para cachear las predicciones recientes del clúster y mitigar latencias en consultas repetitivas.

---

## Cambios Técnicos y Refactorización

Para adaptar el código local al nuevo entorno distribuido, se implementaron las siguientes modificaciones a nivel de código base:

### 1. Desacoplamiento de Modelos de Machine Learning (MongoDB)
En la versión local original, los nodos TCP del backend leían los archivos de modelos (`.json`) directamente desde el sistema de archivos (disco duro local). Para soportar contenedores efímeros (como los de Render):
*   Se introdujo una interfaz de abstracción llamada `ProveedorModelos` en el paquete `cluster`.
*   El paquete `db` (`ClienteMongo`) pasó a implementar dicha interfaz, añadiendo lógica para recuperar los modelos completos directamente desde una nueva colección en MongoDB llamada `ml_models`.
*   Se desarrolló un script interno integrado al servidor (`--upload-models` en `main.go`) que permite sembrar la base de datos leyendo los archivos `.json` generados en la fase de Data Science e insertándolos directamente a Mongo Atlas.

### 2. Contenerización del Backend (Docker)
El servidor en Go fue empaquetado utilizando un patrón Multi-Stage en Docker:
*   **Stage Builder:** Emplea `golang:alpine` (para dar soporte desde Go 1.25 en adelante). Se encarga de descargar módulos y compilar el binario del backend de forma estática (`CGO_ENABLED=0`).
*   **Stage Runner:** Emplea `alpine:latest`. Se instaló el paquete `ca-certificates` para garantizar el soporte de conexiones cifradas (TLS) requeridas por MongoDB Atlas y Upstash Redis.

### 3. Infraestructura como Código (IoC)
Se agregaron archivos de configuración declarativos para integrar los servicios de alojamiento automático (CI/CD):
*   **`render.yaml`:** Especifica a la plataforma Render que debe crear un *Web Service* utilizando el entorno Docker, apuntando a `backend/Dockerfile` como origen, y definiendo la necesidad de inyectar las variables de entorno `MONGO_URI` y `REDIS_URI` durante el arranque.
*   **`netlify.toml`:** Especifica a Netlify el comando de compilación (`npm run build`), el directorio de publicación (`dist`) y una regla de redirección universal (`/* -> /index.html`) indispensable para que el router de la Single Page Application (SPA) funcione correctamente.

### 4. Enrutamiento Dinámico en el Frontend
*   El cliente HTTP/WS en `frontend/src/api.js` fue reescrito para abandonar el *hardcoding* de `localhost`.
*   Ahora extrae la URL del backend desde el inyector de entorno de Vite (`import.meta.env.VITE_API_URL`). 
*   Se implementó una expresión regular inteligente que convierte automáticamente el protocolo HTTP/HTTPS de la API base a protocolos WS/WSS para habilitar los WebSockets bajo entornos de producción seguros.
