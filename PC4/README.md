# SecurityGO PC4 — API REST + Cluster ML + MongoDB

Extensión distribuida de SecurityGO (PC3) con API REST, cluster de nodos ML y persistencia en MongoDB.

## Requisitos previos

- Go 1.21+
- MongoDB corriendo en `localhost:27017`
- Modelos entrenados del PC3 en `../models/`

## Pasos para ejecutar

### 1. Entrar a la carpeta PC4
```powershell
cd PC4
```

### 2. Descargar dependencias
```powershell
go mod tidy
```

### 3. Ejecutar el servidor
```powershell
go run .
```

El servidor queda corriendo en `http://localhost:8080`

## Probar los endpoints

### Modelo 1 — Tipo de crimen
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/predict/crime-type" `
  -Method POST -ContentType "application/json" `
  -Body '{"hour":12,"day_of_week":1,"month":6,"area":1,"premis_cd":101,"part_1_2":1,"victim_identified":1,"days_to_report":0}'
```

### Modelo 2 — Zona de riesgo
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/predict/risk-zone" `
  -Method POST -ContentType "application/json" `
  -Body '{"hour":12,"day_of_week":1,"month":6,"crm_cd":510,"premis_cd":101,"part_1_2":1,"area":1}'
```

### Modelo 3 — Probabilidad de arresto
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/predict/arrest-prob" `
  -Method POST -ContentType "application/json" `
  -Body '{"crm_cd":101,"area":1,"hour":12,"day_of_week":1,"premis_cd":6,"weapon_present":0,"victim_identified":1,"days_to_report":3,"part_1_2":1}'
```

### Estado del cluster
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/health"
```

### Historial de predicciones
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/predictions?model=model1&limit=5"
```
