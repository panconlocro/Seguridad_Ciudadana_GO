# run_backend.ps1 — SecurityGO Backend (TB2)
# Uso: .\run_backend.ps1 [--port 8080] [--mongo mongodb://localhost:27017] [--workers 2]
# Ejecutar desde la carpeta backend/ dentro del repo

param(
    [string]$port    = "8080",
    [string]$mongo   = "mongodb://localhost:27017",
    [int]   $workers = 2,
    [string]$model1  = "./models_cache/model1.json",
    [string]$model2  = "./models_cache/model2.json",
    [string]$model3  = "./models_cache/model3.json"
)

Write-Host ""
Write-Host "╔══════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║      SecurityGO Backend — API REST + Cluster ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# Se omitió la verificación estricta de archivos locales de modelos
# porque el servidor los descargará automáticamente desde MongoDB GridFS
# si no existen en ./models_cache/.
Write-Host "[Init] ✔ Iniciando backend. Los modelos se descargarán de GridFS si están ausentes." -ForegroundColor Green

# Verificar MongoDB
Write-Host "[Init] Verificando MongoDB en $mongo ..." -ForegroundColor Yellow

# Descargar dependencias si no existen
if (-not (Test-Path "go.sum")) {
    Write-Host "[Init] Descargando dependencias (go mod tidy)..." -ForegroundColor Yellow
    go mod tidy
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] go mod tidy falló" -ForegroundColor Red
        exit 1
    }
}

Write-Host "[Init] Iniciando servidor en puerto $port con $workers workers por nodo..." -ForegroundColor Green
Write-Host ""

go run . --port $port --mongo $mongo --workers $workers --model1 $model1 --model2 $model2 --model3 $model3