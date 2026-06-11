package main

import (
	"fmt"
)

// ═══════════════════════════════════════════════════════
// MAIN BENCHMARK — Demuestra carga concurrente vs secuencial
// Ejecutar: go run shared.go loader.go main_benchmark.go
// ═══════════════════════════════════════════════════════

func main() {
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  SISTEMA DE PREDICCIÓN CRIMINAL — SEGURIDAD CIUDADANA")
	fmt.Println("  Benchmark: Carga Secuencial vs Concurrente")
	fmt.Println("═══════════════════════════════════════════════════")

	path := "../../data/processed/Crime_Data_Clean.csv"

	// Ejecutar benchmark completo
	datos := BenchmarkCarga(path)

	fmt.Printf("\n✔ Datos listos para entrenamiento: %d registros\n", len(datos))
	fmt.Println("\n► Para entrenar los modelos ejecuta:")
	fmt.Println("  go run shared.go loader.go model1.go")
	fmt.Println("  go run shared.go loader.go model2.go")
	fmt.Println("  go run shared.go loader.go model3.go")
}
