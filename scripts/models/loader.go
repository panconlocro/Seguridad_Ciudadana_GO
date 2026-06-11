package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════
// CARGADOR CONCURRENTE DE DATOS
// Divide el CSV en bloques y procesa cada uno en paralelo
// mediante goroutines y channels
// ═══════════════════════════════════════════════════════

// WorkerResult almacena el resultado de cada goroutine
type WorkerResult struct {
	WorkerID    int
	Registros   []CrimeClean
	Procesados  int
	Descartados int
	Tiempo      time.Duration
}

// leerTodasLasFilas lee el CSV completo y retorna todas las filas raw
func leerTodasLasFilas(path string) ([]string, [][]string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error abriendo archivo: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.ReuseRecord = false

	// Leer encabezado
	headers, err := reader.Read()
	if err != nil {
		log.Fatalf("Error leyendo encabezado: %v", err)
	}

	// Leer todas las filas
	var filas [][]string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		filas = append(filas, row)
	}

	return headers, filas
}

// procesarBloque convierte un bloque de filas raw a CrimeClean
func procesarBloque(workerID int, headers []string, filas [][]string, resultCh chan<- WorkerResult, wg *sync.WaitGroup) {
	defer wg.Done()
	inicio := time.Now()

	// Construir mapa de índices
	idx := make(map[string]int)
	for i, h := range headers {
		idx[h] = i
	}

	get := func(row []string, col string) string {
		if i, ok := idx[col]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	toInt := func(s string) int {
		v, _ := strconv.Atoi(strings.TrimSpace(s))
		return v
	}

	toFloat := func(s string) float64 {
		v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return v
	}

	var registros []CrimeClean
	descartados := 0

	for _, row := range filas {
		// Parsear day_of_week
		dow := dayOfWeekToInt(get(row, "day_of_week"))

		// victim_identified
		victIdent := get(row, "victim_identified") == "true"

		// Binarizar arresto
		statusDesc := strings.ToLower(get(row, "status_desc"))
		arresto := 0
		if strings.Contains(statusDesc, "arrest") {
			arresto = 1
		}

		lat := toFloat(get(row, "lat"))
		lon := toFloat(get(row, "lon"))

		// Filtrar coordenadas inválidas
		if lat == 0 && lon == 0 {
			descartados++
			continue
		}

		c := CrimeClean{
			Hour:             toInt(get(row, "hour")),
			DayOfWeek:        dow,
			Month:            toInt(get(row, "month")),
			Year:             toInt(get(row, "year")),
			DaysToReport:     toInt(get(row, "days_to_report")),
			Area:             toInt(get(row, "area")),
			Lat:              lat,
			Lon:              lon,
			CrmCd:            toInt(get(row, "crm_cd")),
			CrmCdDesc:        get(row, "crm_cd_desc"),
			Part12:           toInt(get(row, "part_1_2")),
			PremisCd:         toInt(get(row, "premis_cd")),
			WeaponDesc:       get(row, "weapon_desc"),
			VictAge:          toInt(get(row, "vict_age")),
			VictSex:          get(row, "vict_sex"),
			VictDescent:      get(row, "vict_descent"),
			VictimIdentified: victIdent,
			Arresto:          arresto,
		}

		registros = append(registros, c)
	}

	resultCh <- WorkerResult{
		WorkerID:    workerID,
		Registros:   registros,
		Procesados:  len(filas),
		Descartados: descartados,
		Tiempo:      time.Since(inicio),
	}
}

// CargarCSVConcurrente carga el CSV dividiéndolo en bloques paralelos
func CargarCSVConcurrente(path string, numWorkers int) []CrimeClean {
	fmt.Printf("\n[Cargador] Iniciando carga concurrente con %d workers...\n", numWorkers)
	fmt.Printf("[Cargador] Leyendo archivo: %s\n", path)

	inicioTotal := time.Now()

	// Paso 1: Leer todas las filas (necesario para dividir en bloques)
	headers, filas := leerTodasLasFilas(path)
	fmt.Printf("[Cargador] ✔ Filas leídas: %d\n", len(filas))

	// Paso 2: Dividir en bloques iguales
	tamBloque := len(filas) / numWorkers
	fmt.Printf("[Cargador] Bloques: %d | Filas por bloque: ~%d\n", numWorkers, tamBloque)

	// Paso 3: Lanzar goroutines — una por bloque
	resultCh := make(chan WorkerResult, numWorkers)
	var wg sync.WaitGroup

	inicioProcesamiento := time.Now()

	for i := 0; i < numWorkers; i++ {
		inicio := i * tamBloque
		fin := inicio + tamBloque
		if i == numWorkers-1 {
			fin = len(filas) // último worker toma el remanente
		}

		wg.Add(1)
		fmt.Printf("[Cargador] ✔ Worker %d lanzado → filas [%d - %d]\n", i+1, inicio, fin-1)
		go procesarBloque(i+1, headers, filas[inicio:fin], resultCh, &wg)
	}

	// Cerrar canal cuando todos los workers terminen
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Paso 4: Recolectar resultados de todos los workers via channel
	var todosLosRegistros []CrimeClean
	totalProcesados := 0
	totalDescartados := 0

	for resultado := range resultCh {
		todosLosRegistros = append(todosLosRegistros, resultado.Registros...)
		totalProcesados += resultado.Procesados
		totalDescartados += resultado.Descartados
		fmt.Printf("[Cargador] Worker %d → %d válidos | %d descartados | tiempo: %v\n",
			resultado.WorkerID,
			len(resultado.Registros),
			resultado.Descartados,
			resultado.Tiempo.Round(time.Millisecond))
	}

	tiempoProcesamiento := time.Since(inicioProcesamiento)
	tiempoTotal := time.Since(inicioTotal)

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("       REPORTE DE CARGA CONCURRENTE")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Workers utilizados     : %d goroutines\n", numWorkers)
	fmt.Printf("Total filas leídas     : %d\n", totalProcesados)
	fmt.Printf("Registros válidos      : %d\n", len(todosLosRegistros))
	fmt.Printf("Registros descartados  : %d\n", totalDescartados)
	fmt.Printf("Tiempo procesamiento   : %v\n", tiempoProcesamiento.Round(time.Millisecond))
	fmt.Printf("Tiempo total (lectura+proceso): %v\n", tiempoTotal.Round(time.Millisecond))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return todosLosRegistros
}

// BenchmarkCarga compara carga secuencial vs concurrente y muestra el speedup
func BenchmarkCarga(path string) []CrimeClean {
	fmt.Println("\n╔══════════════════════════════════════════╗")
	fmt.Println("║     BENCHMARK: SECUENCIAL vs PARALELO    ║")
	fmt.Println("╚══════════════════════════════════════════╝")

	// ── Carga SECUENCIAL
	fmt.Println("\n[Secuencial] Iniciando...")
	t1 := time.Now()
	datosSecuencial := CargarCSVLimpio(path)
	tiempoSecuencial := time.Since(t1)
	fmt.Printf("[Secuencial] ✔ %d registros en %v\n", len(datosSecuencial), tiempoSecuencial.Round(time.Millisecond))

	// ── Carga CONCURRENTE con distintos números de workers
	configuraciones := []int{2, 4, 8}
	var mejorDatos []CrimeClean
	var mejorTiempo time.Duration

	for _, nWorkers := range configuraciones {
		fmt.Printf("\n[Paralelo-%d] Iniciando...\n", nWorkers)
		t2 := time.Now()
		datosParalelo := CargarCSVConcurrente(path, nWorkers)
		tiempoParalelo := time.Since(t2)
		speedup := float64(tiempoSecuencial) / float64(tiempoParalelo)

		fmt.Printf("[Paralelo-%d] ✔ %d registros en %v\n", nWorkers, len(datosParalelo), tiempoParalelo.Round(time.Millisecond))
		fmt.Printf("[Paralelo-%d] ⚡ Speedup vs secuencial: %.2fx\n", nWorkers, speedup)

		if mejorTiempo == 0 || tiempoParalelo < mejorTiempo {
			mejorTiempo = tiempoParalelo
			mejorDatos = datosParalelo
		}
	}

	// ── Tabla resumen
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  RESUMEN BENCHMARK")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Secuencial (1 hilo)  : %v\n", tiempoSecuencial.Round(time.Millisecond))
	for _, nWorkers := range configuraciones {
		t2 := time.Now()
		_ = CargarCSVConcurrente(path, nWorkers)
		tp := time.Since(t2)
		speedup := float64(tiempoSecuencial) / float64(tp)
		fmt.Printf("  Paralelo (%d workers): %v | Speedup: %.2fx\n",
			nWorkers, tp.Round(time.Millisecond), speedup)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return mejorDatos
}
