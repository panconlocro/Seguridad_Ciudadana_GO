# Guía PC3: entrenamiento, evaluación y predicción

## Ubicación de ejecución

Todos los comandos se ejecutan desde la raíz `Seguridad_Ciudadana_GO`, donde
se encuentran `go.mod`, `run_models.sh` y `run_models.ps1`.

macOS o Linux:

```bash
cd /ruta/al/proyecto/Seguridad_Ciudadana_GO
./run_models.sh help
```

Windows PowerShell:

```powershell
cd C:\ruta\al\proyecto\Seguridad_Ciudadana_GO
.\run_models.ps1 help
```

Si PowerShell bloquea scripts, habilitarlos para la terminal actual:

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
```

## Dataset predeterminado

Si no se indica `--input`, la CLI utiliza:

```text
scripts/Cleanse/data/processed/Crime_Data_Clean.csv
```

Otro CSV puede utilizarse con `--input`, pero debe conservar las columnas del
dataset limpio. La CLI valida el esquema y falla con un mensaje claro si
faltan columnas.

Para regenerar el dataset limpio desde el archivo raw:

```bash
go run ./scripts/Cleanse
```

## Modelos implementados

Los tres modelos son variantes de **Random Forest implementadas completamente
en Go**. Cada árbol entrena con una muestra bootstrap y un subconjunto
aleatorio de features.

| Tipo | Objetivo | Métricas |
| --- | --- | --- |
| `model1` | Clasificar el tipo de crimen | Accuracy |
| `model2` | Predecir latitud y longitud | MAE, RMSE y error aproximado en km |
| `model3` | Clasificar probabilidad de arresto | Accuracy, precision, recall, F1 y matriz de confusión |

El entrenamiento usa goroutines, channels y un pool configurable con
`--workers`. Los árboles son trabajos independientes y el proceso principal
combina sus resultados.

## Separación de responsabilidades

- `scripts/models/loader.go`: carga concurrente y partición del CSV.
- `scripts/models/shared.go`: tipos, split train/test y árboles compartidos.
- `scripts/models/model1.go`, `model2.go`, `model3.go`: lógica de cada modelo.
- `scripts/models/persistence.go`: guardado y carga de modelos JSON.
- `scripts/models/main.go`: CLI y flujo de ejecución.
- `scripts/models/models_test.go`: pruebas, concurrencia y reproducibilidad.

## Entrenar y guardar

Entrena utilizando todo el CSV y guarda el modelo como JSON:

```bash
./run_models.sh train \
  --model-type model1 \
  --input scripts/Cleanse/data/processed/Crime_Data_Clean.csv \
  --output models/model1.json \
  --workers 8
```

PowerShell:

```powershell
.\run_models.ps1 train --model-type model1 --input scripts\Cleanse\data\processed\Crime_Data_Clean.csv --output models\model1.json --workers 8
```

## Evaluar un modelo guardado

`evaluate` carga el JSON y calcula métricas sin volver a entrenar:

```bash
./run_models.sh evaluate \
  --model models/model1.json \
  --input scripts/Cleanse/data/processed/Crime_Data_Clean.csv
```

## Realizar una predicción

Las features deben enviarse en el orden documentado dentro del JSON:

```bash
./run_models.sh predict \
  --model models/model1.json \
  --features "12,1,6,1,101,1,1,0"
```

Orden esperado:

| Tipo | Features |
| --- | --- |
| `model1` | `hour, day_of_week, month, area, premis_cd, part_1_2, victim_identified, days_to_report` |
| `model2` | `hour, day_of_week, month, crm_cd, premis_cd, part_1_2, area` |
| `model3` | `crm_cd, area, hour, day_of_week, premis_cd, weapon_present, victim_identified, days_to_report, part_1_2` |

## Pipeline completo

`run` carga el CSV limpio, separa train/test 80/20, entrena, evalúa, muestra
predicciones de ejemplo y guarda el modelo:

```bash
./run_models.sh run --model-type model3 --workers 8 --output models/model3.json
```

Los comandos compatibles `model1`, `model2` y `model3` ejecutan el mismo flujo:

```bash
./run_models.sh model1 --workers 8
./run_models.sh model2 --workers 8
./run_models.sh model3 --workers 8
./run_models.sh all --workers 8
```

## Benchmark secuencial contra paralelo

El benchmark entrena el mismo modelo con la misma semilla y configuración.
Primero usa `1` worker y luego la cantidad indicada:

```bash
./run_models.sh benchmark \
  --model-type model1 \
  --workers 8 \
  --trees 10 \
  --depth 8
```

## Flags disponibles

| Flag | Descripción | Predeterminado |
| --- | --- | --- |
| `--input`, `--data` | CSV limpio | `scripts/Cleanse/data/processed/Crime_Data_Clean.csv` |
| `--model-type` | `model1`, `model2` o `model3` | `model1` |
| `--model` | JSON utilizado por `evaluate` o `predict` | requerido |
| `--output` | Destino del modelo entrenado | `models/<tipo>.json` |
| `--workers` | Workers del pool de árboles | hasta 8 según CPU |
| `--trees` | Cantidad de árboles | `10` |
| `--depth` | Profundidad máxima | `8` |
| `--min-samples` | Muestras mínimas para dividir un nodo | `50` |
| `--seed` | Semilla reproducible | `42` |
| `--features` | Valores para `predict`, separados por comas | requerido |

## Validaciones antes de entregar

```bash
go test ./...
go vet ./...
go test -race ./...
./run_models.sh help
```

Prueba rápida del pipeline:

```bash
./run_models.sh run \
  --model-type model1 \
  --workers 4 \
  --trees 2 \
  --depth 2 \
  --min-samples 1000 \
  --output models/model1-smoke.json
```
