# Etapa de construcción
FROM golang:alpine AS builder

WORKDIR /app

# Copiar dependencias y descargar
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar el binario estáticamente
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o securitygo-backend .

# Etapa final (imagen ligera)
FROM alpine:latest

# Añadir certificados para conexiones HTTPS (Mongo Atlas, Redis)
RUN apk --no-cache add ca-certificates

WORKDIR /app/

# Copiar el binario compilado
COPY --from=builder /app/securitygo-backend .

# Exponer el puerto por defecto (Render inyecta PORT)
EXPOSE 8080

# Comando por defecto
CMD ["./securitygo-backend"]
