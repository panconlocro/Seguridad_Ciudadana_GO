package main

import (
	"fmt"
)

// ═══════════════════════════════════════════════════════
// BENCHMARK — Demuestra carga concurrente vs secuencial
// ═══════════════════════════════════════════════════════

func EjecutarBenchmark(path string) {
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  SISTEMA DE PREDICCIÓN CRIMINAL — SEGURIDAD CIUDADANA")
	fmt.Println("  Benchmark: Carga Secuencial vs Concurrente")
	fmt.Println("═══════════════════════════════════════════════════")

	// Ejecutar benchmark completo
	datos := BenchmarkCarga(path)

	fmt.Printf("\n✔ Datos listos para entrenamiento: %d registros\n", len(datos))
	fmt.Println("\n► Para entrenar los modelos ejecuta:")
	fmt.Println("  ./run_models.sh model1")
	fmt.Println("  ./run_models.sh model2")
	fmt.Println("  ./run_models.sh model3")
	fmt.Println("  En Windows PowerShell, reemplaza ./run_models.sh por .\\run_models.ps1")
}
