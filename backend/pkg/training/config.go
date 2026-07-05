package training

import (
	"fmt"
	"runtime"
)

const maxMuestrasPorArbol = 100000

type ConfigEntrenamiento struct {
	NumArboles  int
	MaxProf     int
	MinMuestras int
	Workers     int
	Seed        int64
}

func ConfigPredeterminada() ConfigEntrenamiento {
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	return ConfigEntrenamiento{
		NumArboles:  10,
		MaxProf:     8,
		MinMuestras: 50,
		Workers:     workers,
		Seed:        42,
	}
}

func (c ConfigEntrenamiento) Validar() error {
	if c.NumArboles < 1 {
		return fmt.Errorf("trees debe ser mayor que cero")
	}
	if c.MaxProf < 1 {
		return fmt.Errorf("depth debe ser mayor que cero")
	}
	if c.MinMuestras < 1 {
		return fmt.Errorf("min-samples debe ser mayor que cero")
	}
	if c.Workers < 1 {
		return fmt.Errorf("workers debe ser mayor que cero")
	}
	return nil
}

func numFeaturesPara(muestras []Muestra) (int, error) {
	if len(muestras) == 0 || len(muestras[0].Features) == 0 {
		return 0, fmt.Errorf("no hay muestras válidas para entrenar")
	}
	return max(1, intSqrt(len(muestras[0].Features))), nil
}

func intSqrt(n int) int {
	resultado := 0
	for (resultado+1)*(resultado+1) <= n {
		resultado++
	}
	return resultado
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
