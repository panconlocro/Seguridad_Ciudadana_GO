package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────
// ESTRUCTURA DEL REGISTRO ORIGINAL (CSV)
// ─────────────────────────────────────────
type CrimeRaw struct {
	DRNO        string // "190326475"
	DateRptd    string // "3/01/2020 00:00"
	DateOcc     string // "3/01/2020 00:00"
	TimeOcc     string // "2130"
	Area        string // "7"
	AreaName    string // "Wilshire"
	RptDistNo   string // "784"
	Part12      string // "1"
	CrmCd       string // "510"
	CrmCdDesc   string // "VEHICLE - STOLEN"
	Mocodes     string // "" o "1822 1402 0344"
	VictAge     string // "0" o "47"
	VictSex     string // "M" o ""
	VictDescent string // "O" o ""
	PremisCd    string // "101"
	PremisDesc  string // "STREET"
	WeaponUsed  string // "" (puede ser nulo)
	WeaponDesc  string // "" (puede ser nulo)
	Status      string // "AA"
	StatusDesc  string // "Adult Arrest"
	CrmCd1      string // "510"
	CrmCd2      string // "998" o ""
	CrmCd3      string // ""
	CrmCd4      string // ""
	Location    string // "1900 S  LONGWOOD AV"
	CrossStreet string // "" (puede ser nulo)
	Lat         string // "34.0375"
	Lon         string // "-118.3506"
}

// ─────────────────────────────────────────
// ESTRUCTURA LIMPIA (para modelos ML)
// ─────────────────────────────────────────
type CrimeClean struct {
	// Temporales
	HoraOcurrencia int // 0–23 (extraído de TimeOcc)
	DiaSemana      int // 0=domingo, 6=sábado
	Mes            int // 1–12

	// Geográficas
	Area int     // 1–21
	Lat  float64 // 33.7 – 34.3
	Lon  float64 // -118.7 – -118.1

	// Tipo de crimen
	CrmCd     int    // código numérico
	CrmCdDesc string // "VEHICLE - STOLEN"
	Part12    int    // 1 o 2

	// Contexto
	PremisCd   int    // código del lugar
	PremisDesc string // "STREET"
	WeaponDesc string // "NO WEAPON" si nulo

	// Víctima
	VictAge     float64 // normalizado min-max
	VictSex     int     // 0=F, 1=M, 2=X
	VictDescent string  // "O", "H", "W", etc.

	// Target Modelo 3
	Arresto int // 1=arresto, 0=no arresto
}

// ─────────────────────────────────────────
// PASO 1: CARGAR CSV
// ─────────────────────────────────────────
func cargarCSV(path string) []CrimeRaw {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error abriendo archivo: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Saltar encabezado
	_, _ = reader.Read()

	var registros []CrimeRaw
	for {
		row, err := reader.Read()
		if err != nil {
			break
		}
		if len(row) < 28 {
			continue // saltar filas incompletas
		}
		registros = append(registros, CrimeRaw{
			DRNO: row[0], DateRptd: row[1], DateOcc: row[2],
			TimeOcc: row[3], Area: row[4], AreaName: row[5],
			RptDistNo: row[6], Part12: row[7], CrmCd: row[8],
			CrmCdDesc: row[9], Mocodes: row[10], VictAge: row[11],
			VictSex: row[12], VictDescent: row[13], PremisCd: row[14],
			PremisDesc: row[15], WeaponUsed: row[16], WeaponDesc: row[17],
			Status: row[18], StatusDesc: row[19], CrmCd1: row[20],
			CrmCd2: row[21], CrmCd3: row[22], CrmCd4: row[23],
			Location: row[24], CrossStreet: row[25], Lat: row[26],
			Lon: row[27],
		})
	}

	fmt.Printf("✔ Registros cargados: %d\n", len(registros))
	return registros
}

// ─────────────────────────────────────────
// PASO 2: TRANSFORMAR UN REGISTRO
// ─────────────────────────────────────────
func transformar(r CrimeRaw, minEdad, maxEdad float64) (CrimeClean, bool) {
	var c CrimeClean

	// ── Hora del incidente (TIME OCC: 2130 → 21)
	timeOcc, err := strconv.Atoi(strings.TrimSpace(r.TimeOcc))
	if err != nil {
		return c, false
	}
	c.HoraOcurrencia = timeOcc / 100

	// ── Día de la semana y mes desde DATE OCC ("3/01/2020 00:00")
	fecha, err := time.Parse("1/02/2006 15:04", strings.TrimSpace(r.DateOcc))
	if err != nil {
		// Intentar formato alternativo
		fecha, err = time.Parse("01/02/2006 15:04", strings.TrimSpace(r.DateOcc))
		if err != nil {
			return c, false
		}
	}
	c.DiaSemana = int(fecha.Weekday()) // 0=domingo
	c.Mes = int(fecha.Month())

	// ── Área policial
	area, err := strconv.Atoi(strings.TrimSpace(r.Area))
	if err != nil || area < 1 || area > 21 {
		return c, false
	}
	c.Area = area

	// ── Coordenadas (eliminar registros con LAT=0 o LON=0)
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(r.Lat), 64)
	lon, err2 := strconv.ParseFloat(strings.TrimSpace(r.Lon), 64)
	if err1 != nil || err2 != nil || math.Abs(lat) < 0.001 || math.Abs(lon) < 0.001 {
		return c, false
	}
	c.Lat = lat
	c.Lon = lon

	// ── Código y descripción del crimen
	crmCd, err := strconv.Atoi(strings.TrimSpace(r.CrmCd))
	if err != nil {
		return c, false
	}
	c.CrmCd = crmCd
	c.CrmCdDesc = strings.TrimSpace(r.CrmCdDesc)

	// ── Part 1-2
	part, _ := strconv.Atoi(strings.TrimSpace(r.Part12))
	c.Part12 = part

	// ── Tipo de lugar
	premisCd, _ := strconv.Atoi(strings.TrimSpace(r.PremisCd))
	c.PremisCd = premisCd
	c.PremisDesc = strings.TrimSpace(r.PremisDesc)
	if c.PremisDesc == "" {
		c.PremisDesc = "UNKNOWN"
	}

	// ── Arma (67.5% nulos → imputar "NO WEAPON")
	c.WeaponDesc = strings.TrimSpace(r.WeaponDesc)
	if c.WeaponDesc == "" {
		c.WeaponDesc = "NO WEAPON"
	}

	// ── Víctima: edad normalizada min-max
	edad, err := strconv.ParseFloat(strings.TrimSpace(r.VictAge), 64)
	if err != nil || edad < 0 || edad > 120 {
		edad = 0
	}
	if maxEdad > minEdad {
		c.VictAge = (edad - minEdad) / (maxEdad - minEdad)
	}

	// ── Víctima: sexo (F=0, M=1, X=2)
	switch strings.ToUpper(strings.TrimSpace(r.VictSex)) {
	case "F":
		c.VictSex = 0
	case "M":
		c.VictSex = 1
	default:
		c.VictSex = 2 // X = desconocido
	}

	// ── Víctima: origen étnico
	c.VictDescent = strings.TrimSpace(r.VictDescent)
	if c.VictDescent == "" {
		c.VictDescent = "X"
	}

	// ── Target Modelo 3: arresto (1) o no (0)
	status := strings.ToLower(strings.TrimSpace(r.StatusDesc))
	if strings.Contains(status, "arrest") {
		c.Arresto = 1
	} else {
		c.Arresto = 0
	}

	return c, true
}

// ─────────────────────────────────────────
// PASO 3: CALCULAR MIN/MAX DE EDAD (paralelo)
// ─────────────────────────────────────────
func calcularMinMaxEdad(registros []CrimeRaw) (float64, float64) {
	minEdad := math.MaxFloat64
	maxEdad := -math.MaxFloat64
	var mu sync.Mutex
	var wg sync.WaitGroup

	numWorkers := 8
	tamParticion := len(registros) / numWorkers

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		inicio := i * tamParticion
		fin := inicio + tamParticion
		if i == numWorkers-1 {
			fin = len(registros)
		}
		go func(parte []CrimeRaw) {
			defer wg.Done()
			localMin, localMax := math.MaxFloat64, -math.MaxFloat64
			for _, r := range parte {
				edad, err := strconv.ParseFloat(strings.TrimSpace(r.VictAge), 64)
				if err != nil || edad < 0 || edad > 120 {
					continue
				}
				if edad < localMin {
					localMin = edad
				}
				if edad > localMax {
					localMax = edad
				}
			}
			mu.Lock()
			if localMin < minEdad {
				minEdad = localMin
			}
			if localMax > maxEdad {
				maxEdad = localMax
			}
			mu.Unlock()
		}(registros[inicio:fin])
	}
	wg.Wait()
	return minEdad, maxEdad
}

// ─────────────────────────────────────────
// PASO 4: LIMPIEZA PARALELA CON GOROUTINES
// ─────────────────────────────────────────
func limpiarParalelo(registros []CrimeRaw, minEdad, maxEdad float64) []CrimeClean {
	numWorkers := 8
	tamParticion := len(registros) / numWorkers

	canal := make(chan []CrimeClean, numWorkers)
	var wg sync.WaitGroup

	inicio := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		ini := i * tamParticion
		fin := ini + tamParticion
		if i == numWorkers-1 {
			fin = len(registros)
		}
		go func(parte []CrimeRaw) {
			defer wg.Done()
			var limpios []CrimeClean
			for _, r := range parte {
				if c, ok := transformar(r, minEdad, maxEdad); ok {
					limpios = append(limpios, c)
				}
			}
			canal <- limpios
		}(registros[ini:fin])
	}

	go func() {
		wg.Wait()
		close(canal)
	}()

	var resultado []CrimeClean
	for particion := range canal {
		resultado = append(resultado, particion...)
	}

	elapsed := time.Since(inicio)
	fmt.Printf("✔ Registros limpios: %d / %d\n", len(resultado), len(registros))
	fmt.Printf("✔ Registros descartados: %d\n", len(registros)-len(resultado))
	fmt.Printf("⏱ Tiempo limpieza paralela (8 workers): %v\n", elapsed)

	return resultado
}

// ─────────────────────────────────────────
// PASO 5: ANÁLISIS EXPLORATORIO
// ─────────────────────────────────────────
func analisisExploratorio(registros []CrimeClean) {
	// Frecuencia de crímenes por hora
	porHora := make(map[int]int)
	// Frecuencia de crímenes por área
	porArea := make(map[int]int)
	// Frecuencia de tipo de crimen (top 10)
	porTipo := make(map[string]int)
	// Conteo de arrestos
	totalArrestos := 0

	for _, r := range registros {
		porHora[r.HoraOcurrencia]++
		porArea[r.Area]++
		porTipo[r.CrmCdDesc]++
		totalArrestos += r.Arresto
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("       ANÁLISIS EXPLORATORIO")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Hora pico
	horaPico, maxCrimenes := 0, 0
	for h, count := range porHora {
		if count > maxCrimenes {
			maxCrimenes = count
			horaPico = h
		}
	}
	fmt.Printf("⚠ Hora pico de crímenes : %02d:00 (%d incidentes)\n", horaPico, maxCrimenes)

	// Área más peligrosa
	areaPico, maxArea := 0, 0
	for a, count := range porArea {
		if count > maxArea {
			maxArea = count
			areaPico = a
		}
	}
	fmt.Printf("⚠ Área más peligrosa    : Área %d (%d incidentes)\n", areaPico, maxArea)

	// Tasa de arrestos
	tasaArrestos := float64(totalArrestos) / float64(len(registros)) * 100
	fmt.Printf("⚠ Tasa de arrestos      : %.1f%%\n", tasaArrestos)
	fmt.Printf("⚠ Total registros limpios: %d\n", len(registros))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// ─────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────
func main() {
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("   SISTEMA DE PREDICCIÓN CRIMINAL")
	fmt.Println("   Limpieza y Análisis de Datos")
	fmt.Println("═══════════════════════════════════════")

	// 1. Cargar CSV
	registros := cargarCSV("Crime_Data_from_2020_to_Present.csv")

	// 2. Calcular min/max de edad en paralelo
	minEdad, maxEdad := calcularMinMaxEdad(registros)
	fmt.Printf("✔ Rango de edad víctimas: %.0f – %.0f años\n", minEdad, maxEdad)

	// 3. Limpieza paralela con 8 goroutines
	limpios := limpiarParalelo(registros, minEdad, maxEdad)

	// 4. Análisis exploratorio
	analisisExploratorio(limpios)
}
