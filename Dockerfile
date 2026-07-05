# ==========================================
# ETAPA 1: Compilar Frontend (Node.js)
# ==========================================
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

# Copiar package.json y package-lock.json (si existe)
COPY frontend/package*.json ./
RUN npm ci || npm install

# Copiar el código fuente del frontend
COPY frontend/ ./
RUN npm run build

# ==========================================
# ETAPA 2: Compilar Backend (Go)
# ==========================================
FROM golang:1.22-alpine AS backend-builder
WORKDIR /app/backend

# Copiar go.mod y descargar dependencias
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

# Copiar el código fuente del backend
COPY backend/ ./

# Traer el frontend compilado de la etapa 1 a la carpeta public del backend
COPY --from=frontend-builder /app/frontend/dist ./public

# Compilar el binario de Go (con optimizaciones para reducir tamaño)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server .

# ==========================================
# ETAPA 3: Imagen Final de Producción
# ==========================================
FROM alpine:3.19
WORKDIR /app

# Instalar dependencias esenciales para conexión HTTPS/Certificados
RUN apk --no-cache add ca-certificates tzdata

# Copiar el binario y el frontend empaquetado de la etapa 2
COPY --from=backend-builder /app/backend/server .
COPY --from=backend-builder /app/backend/public ./public

# Variables de entorno por defecto
ENV PORT=8080
ENV FRONTEND_DIST="./public"
# MONGO_URI y REDIS_URI serán provistos por Render

# Exponer el puerto
EXPOSE 8080

# Comando para ejecutar la app
CMD ["./server"]
