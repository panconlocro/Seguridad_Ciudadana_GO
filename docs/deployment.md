# Guía de Despliegue (Deployment)

Este documento detalla los cambios arquitectónicos y la configuración necesaria para desplegar la aplicación "Seguridad Ciudadana GO" en la nube, separando el backend, frontend y bases de datos.

## 1. Arquitectura de Despliegue

La aplicación está diseñada para ser desplegada en un entorno distribuido:

*   **Frontend (Netlify):** Aplicación SPA construida con Vite/React. Se despliega como archivos estáticos.
*   **Backend (Render):** Servidor y clúster en Go que maneja la API REST, WebSockets (para analíticas en tiempo real) y los nodos TCP de los modelos predictivos.
*   **Base de Datos (MongoDB Atlas):** Almacena usuarios, el historial de predicciones y **los modelos de Machine Learning en formato JSON**.
*   **Caché (Upstash Redis):** Almacena las predicciones recientes para agilizar las respuestas.

---

## 2. Cambios Realizados para el Despliegue

Para lograr que la aplicación funcionara en la nube de forma nativa, se implementaron los siguientes cambios:

### A. Migración de Modelos a MongoDB
Anteriormente, los nodos TCP del backend leían los árboles de los modelos (`.json`) desde el disco local. Esto era un problema en despliegues en la nube sin disco persistente.
*   Se refactorizaron los módulos `backend/cluster/worker.go` y `backend/cluster/tcp_node.go` para recibir una interfaz `ProveedorModelos`.
*   Se implementó `ClienteMongo` en `backend/db/mongo.go` para cumplir con esta interfaz, guardando y obteniendo los modelos desde la colección `ml_models`.
*   **Comando de Inyección:** Se añadió la bandera `--upload-models` a `main.go`. Al iniciar el backend con esta bandera, lee los archivos `.json` locales de la carpeta `models/` y los inyecta en MongoDB.

### B. Preparación del Backend (Docker & Render)
*   Se creó un `backend/Dockerfile` multi-stage:
    *   **Builder:** Compila el binario de Go con `CGO_ENABLED=0` (compilación estática).
    *   **Runner:** Usa la imagen súper ligera `alpine:latest` e instala certificados (`ca-certificates`) que son estrictamente necesarios para conectarse por TLS a MongoDB Atlas.
*   Se creó `render.yaml` (Infraestructura como Código) en la raíz del proyecto para que Render automatice la lectura del Dockerfile, el puerto `8080`, y deje listas las variables de entorno.
*   *Bugfix:* Se actualizó la imagen base a `golang:alpine` para evitar problemas de compatibilidad con versiones de `go 1.25.0` del `go.mod`.

### C. Preparación del Frontend (Netlify)
*   Se actualizó `frontend/src/api.js` para que el `API_BASE` ya no sea estáticamente `http://localhost:8080`, sino que lea dinámicamente la variable de entorno de Vite: `import.meta.env.VITE_API_URL`.
*   El cliente de WebSockets ahora convierte automáticamente cualquier conexión HTTP (incluyendo HTTPS) al protocolo WS/WSS: `API_BASE.replace(/^http/, 'ws') + '/ws'`.
*   Se creó `netlify.toml` definiendo el directorio base (`frontend`), el comando de construcción (`npm run build`), y la regla general de ruteo para evitar errores 404 en navegaciones de React (`/* -> /index.html`).

---

## 3. Instrucciones de Despliegue Manual

Si necesitas volver a levantar el proyecto desde cero en la nube, sigue estos pasos:

### Paso 1: Configurar las Bases de Datos
1.  Crea un clúster en **MongoDB Atlas** y obtén tu Connection String (`MONGO_URI`). Asegúrate de habilitar acceso desde cualquier IP (`0.0.0.0/0`).
2.  Crea una base de datos en **Upstash Redis** y obtén tu Connection String (`REDIS_URI`).

### Paso 2: Subir los Modelos a Mongo
Antes de desplegar el backend, la base de datos debe contener los modelos. En tu máquina local, compila los modelos (si no lo has hecho) y luego envíalos a Atlas:

```bash
cd backend
go run . --mongo "TU_MONGO_URI_AQUI" --upload-models
```

### Paso 3: Desplegar el Backend en Render
1.  Conecta tu repositorio en GitHub a [Render.com](https://render.com).
2.  Crea un nuevo **Web Service**. (Opcionalmente, Render detectará el `render.yaml` como un Blueprint).
3.  Asegúrate de configurar el *Root Directory* como `backend` y el entorno como *Docker*.
4.  Agrega las variables de entorno:
    *   `MONGO_URI`
    *   `REDIS_URI`
5.  Despliega y copia la URL generada (ej. `https://mi-backend.onrender.com`).

### Paso 4: Desplegar el Frontend en Netlify
1.  Conecta tu repositorio a [Netlify](https://netlify.com).
2.  Netlify leerá automáticamente el archivo `netlify.toml`.
3.  Agrega la siguiente variable de entorno antes del build:
    *   `VITE_API_URL` = `https://mi-backend.onrender.com` (Sin el `/` al final).
4.  Despliega el sitio web.

¡El sistema ahora está corriendo de forma nativa en la nube, utilizando WebSockets seguros (WSS) y modelos pre-cacheados desde MongoDB!
