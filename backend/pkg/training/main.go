package training

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultDataPath = "data/processed/Crime_Data_Clean.csv"

type opcionesCLI struct {
	dataPath   string
	modelType  string
	modelPath  string
	outputPath string
	features   string
	cfg        ConfigEntrenamiento
}

func parsearOpciones(command string, args []string) (opcionesCLI, error) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cfg := ConfigPredeterminada()
	opciones := opcionesCLI{cfg: cfg, modelType: "model1"}
	fs.StringVar(&opciones.dataPath, "data", defaultDataPath, "ruta al CSV limpio")
	fs.StringVar(&opciones.dataPath, "input", defaultDataPath, "alias de --data")
	fs.StringVar(&opciones.modelType, "model-type", "model1", "tipo: model1, model2 o model3")
	fs.StringVar(&opciones.modelType, "type", "model1", "alias de --model-type")
	fs.StringVar(&opciones.modelPath, "model", "", "ruta a un modelo JSON guardado")
	fs.StringVar(&opciones.outputPath, "output", "", "ruta donde guardar el modelo JSON")
	fs.StringVar(&opciones.features, "features", "", "features numéricas separadas por comas")
	fs.IntVar(&opciones.cfg.NumArboles, "trees", cfg.NumArboles, "cantidad de árboles por modelo")
	fs.IntVar(&opciones.cfg.MaxProf, "depth", cfg.MaxProf, "profundidad máxima de cada árbol")
	fs.IntVar(&opciones.cfg.MinMuestras, "min-samples", cfg.MinMuestras, "mínimo de muestras para dividir un nodo")
	fs.IntVar(&opciones.cfg.Workers, "workers", cfg.Workers, "workers del pool de árboles")
	fs.Int64Var(&opciones.cfg.Seed, "seed", cfg.Seed, "semilla aleatoria reproducible")

	if err := fs.Parse(args); err != nil {
		return opcionesCLI{}, err
	}
	if fs.NArg() != 0 {
		return opcionesCLI{}, fmt.Errorf("argumentos no reconocidos: %v", fs.Args())
	}
	if err := opciones.cfg.Validar(); err != nil {
		return opcionesCLI{}, err
	}
	if opciones.modelType != "model1" && opciones.modelType != "model2" && opciones.modelType != "model3" {
		return opcionesCLI{}, fmt.Errorf("model-type debe ser model1, model2 o model3")
	}
	return opciones, nil
}

func imprimirUso() {
	fmt.Fprintln(os.Stderr, "Uso:")
	fmt.Fprintln(os.Stderr, "  ./run_models.sh <comando> [opciones]")
	fmt.Fprintln(os.Stderr, `  .\run_models.ps1 <comando> [opciones]`)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Comandos PC3:")
	fmt.Fprintln(os.Stderr, "  train       Entrena con todo el CSV y guarda el modelo")
	fmt.Fprintln(os.Stderr, "  evaluate    Carga un modelo guardado y calcula métricas")
	fmt.Fprintln(os.Stderr, "  predict     Carga un modelo guardado y predice features")
	fmt.Fprintln(os.Stderr, "  run         Ejecuta train/test, evaluación, predicción y guardado")
	fmt.Fprintln(os.Stderr, "  benchmark   Compara entrenamiento con 1 worker vs N workers")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Comandos compatibles: model1, model2, model3, all")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Opciones principales:")
	fmt.Fprintf(os.Stderr, "  --input PATH        CSV limpio (default: %s)\n", defaultDataPath)
	fmt.Fprintln(os.Stderr, "  --model-type TYPE   model1, model2 o model3 (default: model1)")
	fmt.Fprintln(os.Stderr, "  --model PATH        modelo JSON para evaluate/predict")
	fmt.Fprintln(os.Stderr, "  --output PATH       destino del modelo entrenado")
	fmt.Fprintln(os.Stderr, "  --workers N         workers del pool de árboles")
	fmt.Fprintln(os.Stderr, "  --trees N           cantidad de árboles")
	fmt.Fprintln(os.Stderr, "  --depth N           profundidad máxima")
	fmt.Fprintln(os.Stderr, "  --min-samples N     muestras mínimas por división")
	fmt.Fprintln(os.Stderr, "  --seed N            semilla reproducible")
	fmt.Fprintln(os.Stderr, `  --features "..."    valores separados por comas para predict`)
}

func cargarDatos(path string) ([]CrimeClean, error) {
	fmt.Printf("Cargando datos desde %s\n", path)
	return CargarCSVLimpioE(path)
}

func rutaModeloPredeterminada(tipo string) string {
	return filepath.Join("models", tipo+".json")
}

func entrenarModelo(tipo string, datos []CrimeClean, cfg ConfigEntrenamiento) (*ModeloPersistido, error) {
	artefacto := &ModeloPersistido{
		Version:       versionModelo,
		Tipo:          tipo,
		Algoritmo:     "Random Forest implementado en Go",
		EntrenadoEn:   time.Now().UTC(),
		Configuracion: cfg,
	}
	var err error
	switch tipo {
	case "model1":
		artefacto.Features = featuresModelo1
		artefacto.Modelo1, err = EntrenarModelo1ConConfig(datos, cfg)
	case "model2":
		artefacto.Features = featuresModelo2
		artefacto.Modelo2, err = EntrenarModelo2ConConfig(datos, cfg)
	case "model3":
		artefacto.Features = featuresModelo3
		artefacto.Modelo3, err = EntrenarModelo3ConConfig(datos, cfg)
	}
	return artefacto, err
}

func ejecutarTrain(opciones opcionesCLI) error {
	datos, err := cargarDatos(opciones.dataPath)
	if err != nil {
		return err
	}
	modelo, err := entrenarModelo(opciones.modelType, datos, opciones.cfg)
	if err != nil {
		return err
	}
	output := opciones.outputPath
	if output == "" {
		output = rutaModeloPredeterminada(opciones.modelType)
	}
	return GuardarModelo(output, modelo)
}

// EntrenarDesdeMemoria entrena un modelo a partir de CSV data en memoria y retorna su representación JSON
func EntrenarDesdeMemoria(tipo string, csvData []byte) ([]byte, error) {
	datos, err := CargarCSVLimpioDesdeBytesE(csvData)
	if err != nil {
		return nil, fmt.Errorf("error cargando datos CSV: %w", err)
	}
	
	// Configuracion predeterminada para API
	cfg := ConfigPredeterminada()
	// Ajustar a algo razonable para servidor
	cfg.Workers = 2 
	
	modelo, err := entrenarModelo(tipo, datos, cfg)
	if err != nil {
		return nil, fmt.Errorf("error entrenando modelo: %w", err)
	}
	
	jsonBytes, err := json.MarshalIndent(modelo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializando modelo a JSON: %w", err)
	}
	
	return jsonBytes, nil
}

func ejecutarEvaluate(opciones opcionesCLI) error {
	if opciones.modelPath == "" {
		return fmt.Errorf("evaluate requiere --model PATH")
	}
	modelo, err := CargarModelo(opciones.modelPath)
	if err != nil {
		return err
	}
	datos, err := cargarDatos(opciones.dataPath)
	if err != nil {
		return err
	}
	fmt.Printf("Evaluando %s sobre %d registros\n", modelo.Tipo, len(datos))
	switch modelo.Tipo {
	case "model1":
		modelo.Modelo1.EvaluarMetricas(prepararMuestrasModelo1(datos))
	case "model2":
		modelo.Modelo2.EvaluarMetricas(prepararMuestrasModelo2(datos))
	case "model3":
		modelo.Modelo3.EvaluarMetricas(prepararMuestrasModelo3(datos))
	}
	return nil
}

func parsearFeatures(texto string) ([]float64, error) {
	if strings.TrimSpace(texto) == "" {
		return nil, fmt.Errorf("predict requiere --features con valores separados por comas")
	}
	partes := strings.Split(texto, ",")
	features := make([]float64, len(partes))
	for i, parte := range partes {
		valor, err := strconv.ParseFloat(strings.TrimSpace(parte), 64)
		if err != nil {
			return nil, fmt.Errorf("feature %d inválida %q: %w", i+1, parte, err)
		}
		features[i] = valor
	}
	return features, nil
}

func ejecutarPredict(opciones opcionesCLI) error {
	if opciones.modelPath == "" {
		return fmt.Errorf("predict requiere --model PATH")
	}
	modelo, err := CargarModelo(opciones.modelPath)
	if err != nil {
		return err
	}
	features, err := parsearFeatures(opciones.features)
	if err != nil {
		return err
	}
	if len(features) != len(modelo.Features) {
		return fmt.Errorf("%s requiere %d features (%s), se recibieron %d",
			modelo.Tipo, len(modelo.Features), strings.Join(modelo.Features, ", "), len(features))
	}

	switch modelo.Tipo {
	case "model1":
		clase, confianza := modelo.Modelo1.PredecirConConfianza(features)
		fmt.Printf("Predicción: %s | Confianza: %.2f%%\n", clase, confianza)
	case "model2":
		lat, lon := modelo.Modelo2.Predecir(features)
		fmt.Printf("Predicción: lat=%.6f lon=%.6f\n", lat, lon)
	case "model3":
		clase, prob := modelo.Modelo3.Predecir(features)
		fmt.Printf("Predicción: %s | Probabilidad de arresto: %.2f%%\n", clase, prob*100)
	}
	return nil
}

func ejecutarRun(opciones opcionesCLI, consultar bool) error {
	datos, err := cargarDatos(opciones.dataPath)
	if err != nil {
		return err
	}
	return ejecutarRunConDatos(opciones, datos, consultar)
}

func ejecutarRunConDatos(opciones opcionesCLI, datos []CrimeClean, consultar bool) error {
	var artefacto *ModeloPersistido
	switch opciones.modelType {
	case "model1":
		modelo, _, err := EjecutarPipelineModelo1(datos, opciones.cfg, consultar)
		if err != nil {
			return err
		}
		artefacto = &ModeloPersistido{Modelo1: modelo}
	case "model2":
		modelo, _, err := EjecutarPipelineModelo2(datos, opciones.cfg, consultar)
		if err != nil {
			return err
		}
		artefacto = &ModeloPersistido{Modelo2: modelo}
	case "model3":
		modelo, _, err := EjecutarPipelineModelo3(datos, opciones.cfg, consultar)
		if err != nil {
			return err
		}
		artefacto = &ModeloPersistido{Modelo3: modelo}
	}
	artefacto.Version = versionModelo
	artefacto.Tipo = opciones.modelType
	artefacto.Algoritmo = "Random Forest implementado en Go"
	artefacto.EntrenadoEn = time.Now().UTC()
	artefacto.Configuracion = opciones.cfg
	switch opciones.modelType {
	case "model1":
		artefacto.Features = featuresModelo1
	case "model2":
		artefacto.Features = featuresModelo2
	case "model3":
		artefacto.Features = featuresModelo3
	}
	output := opciones.outputPath
	if output == "" {
		output = rutaModeloPredeterminada(opciones.modelType)
	}
	return GuardarModelo(output, artefacto)
}

func ejecutarAll(opciones opcionesCLI) error {
	datos, err := cargarDatos(opciones.dataPath)
	if err != nil {
		return err
	}
	for _, tipo := range []string{"model1", "model2", "model3"} {
		actual := opciones
		actual.modelType = tipo
		actual.outputPath = rutaModeloPredeterminada(tipo)
		if err := ejecutarRunConDatos(actual, datos, true); err != nil {
			return err
		}
	}
	return nil
}

func run() error {
	if len(os.Args) < 2 {
		imprimirUso()
		return fmt.Errorf("falta un comando")
	}
	command := os.Args[1]
	if command == "help" || command == "-h" || command == "--help" {
		imprimirUso()
		return nil
	}
	opciones, err := parsearOpciones(command, os.Args[2:])
	if err != nil {
		return err
	}

	switch command {
	case "train":
		return ejecutarTrain(opciones)
	case "evaluate":
		return ejecutarEvaluate(opciones)
	case "predict":
		return ejecutarPredict(opciones)
	case "run":
		return ejecutarRun(opciones, true)
	case "benchmark":
		datos, err := cargarDatos(opciones.dataPath)
		if err != nil {
			return err
		}
		return BenchmarkEntrenamiento(datos, opciones.modelType, opciones.cfg)
	case "model1", "model2", "model3":
		opciones.modelType = command
		return ejecutarRun(opciones, true)
	case "all":
		return ejecutarAll(opciones)
	default:
		return fmt.Errorf("comando desconocido %q", command)
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		imprimirUso()
		os.Exit(2)
	}
}
