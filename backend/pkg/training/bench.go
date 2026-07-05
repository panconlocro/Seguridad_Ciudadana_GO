package training

import (
	"fmt"
	"time"
)

func medirEntrenamiento(datos []CrimeClean, tipo string, cfg ConfigEntrenamiento) (time.Duration, error) {
	switch tipo {
	case "model1":
		_, duracion, err := entrenarModelo1Muestras(prepararMuestrasModelo1(datos), cfg)
		return duracion, err
	case "model2":
		_, duracion, err := entrenarModelo2Muestras(prepararMuestrasModelo2(datos), cfg)
		return duracion, err
	case "model3":
		_, duracion, err := entrenarModelo3Muestras(prepararMuestrasModelo3(datos), cfg)
		return duracion, err
	default:
		return 0, fmt.Errorf("tipo de modelo desconocido: %s", tipo)
	}
}

func BenchmarkEntrenamiento(datos []CrimeClean, tipo string, cfg ConfigEntrenamiento) error {
	if cfg.Workers < 2 {
		return fmt.Errorf("benchmark requiere --workers mayor que 1 para comparar contra ejecución secuencial")
	}
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  BENCHMARK DE ENTRENAMIENTO: SECUENCIAL VS PARALELO")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("Modelo: %s | Registros: %d | Workers paralelos: %d\n", tipo, len(datos), cfg.Workers)

	secuencial := cfg
	secuencial.Workers = 1
	tiempoSecuencial, err := medirEntrenamiento(datos, tipo, secuencial)
	if err != nil {
		return err
	}
	tiempoParalelo, err := medirEntrenamiento(datos, tipo, cfg)
	if err != nil {
		return err
	}
	speedup := float64(tiempoSecuencial) / float64(tiempoParalelo)

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Secuencial (1 worker)   : %v\n", tiempoSecuencial)
	fmt.Printf("Paralelo (%d workers)  : %v\n", cfg.Workers, tiempoParalelo)
	fmt.Printf("Speedup                 : %.2fx\n", speedup)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}
