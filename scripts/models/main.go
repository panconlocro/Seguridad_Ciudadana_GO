package main

import (
	"flag"
	"fmt"
	"os"
)

const defaultDataPath = "scripts/Cleanse/data/processed/Crime_Data_Clean.csv"

type opcionesEntrenamiento struct {
	dataPath    string
	numArboles  int
	maxProf     int
	minMuestras int
}

func parsearOpciones(command string, args []string) (opcionesEntrenamiento, error) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	opciones := opcionesEntrenamiento{}
	fs.StringVar(&opciones.dataPath, "data", defaultDataPath, "ruta al CSV limpio")
	fs.IntVar(&opciones.numArboles, "trees", 10, "cantidad de árboles por modelo")
	fs.IntVar(&opciones.maxProf, "depth", 8, "profundidad máxima de cada árbol")
	fs.IntVar(&opciones.minMuestras, "min-samples", 50, "mínimo de muestras para dividir un nodo")

	if err := fs.Parse(args); err != nil {
		return opcionesEntrenamiento{}, err
	}
	if fs.NArg() != 0 {
		return opcionesEntrenamiento{}, fmt.Errorf("argumentos no reconocidos: %v", fs.Args())
	}
	if opciones.numArboles < 1 || opciones.maxProf < 1 || opciones.minMuestras < 1 {
		return opcionesEntrenamiento{}, fmt.Errorf("trees, depth y min-samples deben ser mayores que cero")
	}
	return opciones, nil
}

func imprimirUso() {
	fmt.Fprintln(os.Stderr, "Uso:")
	fmt.Fprintln(os.Stderr, "  macOS/Linux:       ./run_models.sh <comando> [opciones]")
	fmt.Fprintln(os.Stderr, `  Windows PowerShell: .\run_models.ps1 <comando> [opciones]`)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Comandos:")
	fmt.Fprintln(os.Stderr, "  model1      Entrena y consulta el clasificador de tipo de crimen")
	fmt.Fprintln(os.Stderr, "  model2      Entrena y consulta el predictor de zona de riesgo")
	fmt.Fprintln(os.Stderr, "  model3      Entrena y consulta el predictor de arresto")
	fmt.Fprintln(os.Stderr, "  all         Entrena y consulta los tres modelos")
	fmt.Fprintln(os.Stderr, "  benchmark   Compara la carga secuencial y concurrente")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Opciones:")
	fmt.Fprintf(os.Stderr, "  --data PATH         CSV limpio (default: %s)\n", defaultDataPath)
	fmt.Fprintln(os.Stderr, "  --trees N           Cantidad de árboles (default: 10)")
	fmt.Fprintln(os.Stderr, "  --depth N           Profundidad máxima (default: 8)")
	fmt.Fprintln(os.Stderr, "  --min-samples N     Muestras mínimas por división (default: 50)")
}

func cargarDatos(path string) []CrimeClean {
	fmt.Printf("Cargando datos desde %s\n", path)
	return CargarCSVLimpio(path)
}

func main() {
	if len(os.Args) < 2 {
		imprimirUso()
		os.Exit(2)
	}

	command := os.Args[1]
	if command == "help" || command == "-h" || command == "--help" {
		imprimirUso()
		return
	}

	opciones, err := parsearOpciones(command, os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		imprimirUso()
		os.Exit(2)
	}

	switch command {
	case "model1":
		EjecutarModelo1(cargarDatos(opciones.dataPath), opciones.numArboles, opciones.maxProf, opciones.minMuestras)
	case "model2":
		EjecutarModelo2(cargarDatos(opciones.dataPath), opciones.numArboles, opciones.maxProf, opciones.minMuestras)
	case "model3":
		EjecutarModelo3(cargarDatos(opciones.dataPath), opciones.numArboles, opciones.maxProf, opciones.minMuestras)
	case "all":
		datos := cargarDatos(opciones.dataPath)
		EjecutarModelo1(datos, opciones.numArboles, opciones.maxProf, opciones.minMuestras)
		EjecutarModelo2(datos, opciones.numArboles, opciones.maxProf, opciones.minMuestras)
		EjecutarModelo3(datos, opciones.numArboles, opciones.maxProf, opciones.minMuestras)
	case "benchmark":
		EjecutarBenchmark(opciones.dataPath)
	default:
		fmt.Fprintf(os.Stderr, "Error: comando desconocido %q\n\n", command)
		imprimirUso()
		os.Exit(2)
	}
}
