#!/bin/bash
set -e

echo "=== 1. Construyendo el Frontend (React) ==="
cd frontend
npm install
npm run build
cd ..

echo "=== 2. Moviendo Frontend al Backend ==="
# Asegurarnos de que el backend pueda servir el frontend compilado
rm -rf backend/public
mkdir -p backend/public
cp -r frontend/dist/* backend/public/

echo "=== 3. Construyendo el Backend (Go) ==="
cd backend
go build -o server .
cd ..

echo "=== Build Completo ==="
