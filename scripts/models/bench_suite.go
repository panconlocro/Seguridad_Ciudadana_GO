package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parsearListaWorkers(texto string) ([]int, error) {
	partes := strings.Split(texto, ",")
	workers := make([]int, 0, len(partes))
	vistos := make(map[int]bool)
	for _, parte := range partes {
		parte = strings.TrimSpace(parte)
		if parte == "" {
			continue
		}
		n, err := strconv.Atoi(parte)
		if err != nil {
			return nil, fmt.Errorf("worker inválido %q: %w", parte, err)
		}
		if n < 2 {
			return nil, fmt.Errorf("benchmark-suite solo acepta workers paralelos >= 2; el secuencial se mide automáticamente")
		}
		if !vistos[n] {
			workers = append(workers, n)
			vistos[n] = true
		}
	}
	if len(workers) == 0 {
		return nil, fmt.Errorf("worker-list debe incluir al menos un valor >= 2")
	}
	return workers, nil
}

func BenchmarkEntrenamientoSuite(datos []CrimeClean, tipo string, cfg ConfigEntrenamiento, workers []int) error {
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  BENCHMARK SUITE: 1 WORKER VS VARIOS WORKERS")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("Modelo: %s | Registros: %d\n", tipo, len(datos))
	fmt.Printf("Config: arboles=%d | profundidad=%d | min_muestras=%d | muestras_arbol=%d | seed=%d\n",
		cfg.NumArboles, cfg.MaxProf, cfg.MinMuestras, maxMuestrasPorArbol, cfg.Seed)

	secuencial := cfg
	secuencial.Workers = 1
	tiempoSecuencial, err := medirEntrenamiento(datos, tipo, secuencial)
	if err != nil {
		return err
	}

	type filaBenchmark struct {
		workers  int
		duracion time.Duration
		speedup  float64
	}
	filas := make([]filaBenchmark, 0, len(workers))
	for _, n := range workers {
		paralelo := cfg
		paralelo.Workers = n
		duracion, err := medirEntrenamiento(datos, tipo, paralelo)
		if err != nil {
			return err
		}
		filas = append(filas, filaBenchmark{
			workers:  n,
			duracion: duracion,
			speedup:  float64(tiempoSecuencial) / float64(duracion),
		})
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Secuencial (1 worker) : %v\n", tiempoSecuencial)
	fmt.Println("Workers | Tiempo paralelo | Speedup")
	for _, fila := range filas {
		fmt.Printf("%7d | %15v | %.2fx\n", fila.workers, fila.duracion, fila.speedup)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}
