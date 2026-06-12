# Guía para probar los modelos

## ¿Desde dónde ejecutar los comandos?

Los comandos de esta guía deben ejecutarse desde la **raíz del proyecto**.
La raíz es la carpeta `Seguridad_Ciudadana_GO`, donde se encuentran los
archivos `go.mod`, `run_models.sh` y `run_models.ps1`.

Primero, entrar a esa carpeta. En macOS o Linux:

```bash
cd /ruta/al/proyecto/Seguridad_Ciudadana_GO
```

En Windows PowerShell:

```powershell
cd C:\ruta\al\proyecto\Seguridad_Ciudadana_GO
```

Para comprobar que se está en la carpeta correcta en macOS o Linux:

```bash
pwd
ls go.mod run_models.sh
```

En Windows PowerShell:

```powershell
Get-Location
Get-Item go.mod, run_models.ps1
```

Después de entrar a la raíz, en macOS o Linux se pueden ejecutar comandos como:

```bash
go test ./...
go vet ./...
./run_models.sh model1
```

En Windows PowerShell:

```powershell
go test ./...
go vet ./...
.\run_models.ps1 model1
```

- `go test ./...` comprueba que todo el código Go compile correctamente.
- `go vet ./...` busca posibles errores o usos sospechosos en el código Go.
- `./run_models.sh model1` o `.\run_models.ps1 model1` carga los datos,
  entrena el modelo 1 y muestra sus resultados.

## Requisitos

- Go 1.21 o una versión posterior.
- Por defecto, el archivo limpio debe existir en:
  `scripts/Cleanse/data/processed/Crime_Data_Clean.csv`.

Verificar la instalación de Go y mostrar la ayuda del ejecutor:

```bash
go version
./run_models.sh help
```

En Windows PowerShell, mostrar la ayuda:

```powershell
go version
.\run_models.ps1 help
```

Si PowerShell bloquea la ejecución de scripts, habilitarlos solamente para la
terminal actual:

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
```

## Probar que el código compila

```bash
go test ./...
go vet ./...
```

## Prueba rápida

Para comprobar un modelo sin utilizar su configuración completa, se puede
entrenar un solo árbol de poca profundidad:

```bash
./run_models.sh model1 --trees 1 --depth 2 --min-samples 1000
./run_models.sh model2 --trees 1 --depth 2 --min-samples 1000
./run_models.sh model3 --trees 1 --depth 2 --min-samples 1000
```

Estas pruebas siguen leyendo todo el CSV, pero reducen considerablemente el
tiempo de entrenamiento.

En Windows PowerShell, reemplazar `./run_models.sh` por `.\run_models.ps1`:

```powershell
.\run_models.ps1 model1 --trees 1 --depth 2 --min-samples 1000
.\run_models.ps1 model2 --trees 1 --depth 2 --min-samples 1000
.\run_models.ps1 model3 --trees 1 --depth 2 --min-samples 1000
```

## Ejecutar cada modelo

### Modelo 1: clasificación del tipo de crimen

```bash
./run_models.sh model1
```

### Modelo 2: predicción de zona de riesgo

```bash
./run_models.sh model2
```

### Modelo 3: probabilidad de arresto

```bash
./run_models.sh model3
```

En Windows PowerShell:

```powershell
.\run_models.ps1 model1
.\run_models.ps1 model2
.\run_models.ps1 model3
```

## Ejecutar todos los modelos

El comando `all` carga el CSV una sola vez y luego entrena los tres modelos:

```bash
./run_models.sh all
```

En Windows PowerShell:

```powershell
.\run_models.ps1 all
```

## Ejecutar el benchmark de carga

Compara la carga secuencial del CSV con la carga concurrente:

```bash
./run_models.sh benchmark
```

En Windows PowerShell:

```powershell
.\run_models.ps1 benchmark
```

## Opciones de entrenamiento

Todos los comandos aceptan las siguientes opciones:

| Opción | Descripción | Valor predeterminado |
| --- | --- | --- |
| `--data PATH` | Ruta del CSV limpio | `scripts/Cleanse/data/processed/Crime_Data_Clean.csv` |
| `--trees N` | Cantidad de árboles por modelo | `10` |
| `--depth N` | Profundidad máxima de cada árbol | `8` |
| `--min-samples N` | Muestras mínimas para dividir un nodo | `50` |

Ejemplo con una configuración personalizada:

```bash
./run_models.sh model1 --trees 5 --depth 6 --min-samples 100
```

En Windows PowerShell:

```powershell
.\run_models.ps1 model1 --trees 5 --depth 6 --min-samples 100
```

Ejemplo utilizando otro archivo CSV limpio:

```bash
./run_models.sh model3 --data ruta/al/archivo.csv
```

En Windows PowerShell:

```powershell
.\run_models.ps1 model3 --data ruta\al\archivo.csv
```

## ¿Qué CSV utilizan los modelos?

Si no se especifica `--data`, todos los modelos utilizan automáticamente:

```text
scripts/Cleanse/data/processed/Crime_Data_Clean.csv
```

Por ejemplo, este comando utiliza el CSV predeterminado:

```bash
./run_models.sh model1
```

También se puede entrenar un modelo con otro CSV usando `--data`:

```bash
./run_models.sh model1 --data datos/otro_archivo_limpio.csv
```

En Windows PowerShell:

```powershell
.\run_models.ps1 model1 --data datos\otro_archivo_limpio.csv
```

La ruta relativa se interpreta desde la raíz del proyecto. También se puede
usar una ruta absoluta:

```bash
./run_models.sh model1 --data /ruta/completa/otro_archivo_limpio.csv
```

El CSV alternativo debe tener el mismo formato que `Crime_Data_Clean.csv`,
incluyendo las columnas limpias que necesitan los modelos, como `hour`,
`day_of_week`, `month`, `area`, `crm_cd`, `crm_cd_desc`, `premis_cd`,
`part_1_2`, `lat`, `lon`, `weapon_desc`, `victim_identified`,
`days_to_report` y `status_desc`.
