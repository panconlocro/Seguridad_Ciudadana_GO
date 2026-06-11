package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ═══════════════════════════════════════════════════════
// ESTRUCTURA DE REGISTRO LIMPIO (desde Crime_Data_Clean.csv)
// ═══════════════════════════════════════════════════════

type CrimeClean struct {
	// Temporales (engineered)
	Hour         int // 0–23
	DayOfWeek    int // 0=Sunday … 6=Saturday
	Month        int // 1–12
	Year         int // 2020–present
	DaysToReport int // días entre ocurrencia y reporte

	// Geográficas
	Area int     // 1–21
	Lat  float64 // coordenada latitud
	Lon  float64 // coordenada longitud

	// Crimen
	CrmCd     int    // código numérico del crimen
	CrmCdDesc string // "VEHICLE - STOLEN"
	Part12    int    // 1=grave, 2=menos grave

	// Contexto
	PremisCd   int    // código del tipo de lugar
	WeaponDesc string // "NO WEAPON" si no hubo arma

	// Víctima
	VictAge          int    // edad (0 si desconocida)
	VictSex          string // "M", "F", "X"
	VictDescent      string // "H", "W", "B", etc.
	VictimIdentified bool   // true si víctima fue identificada

	// Target Modelo 3
	Arresto int // 1=arresto, 0=no arresto
}

// ═══════════════════════════════════════════════════════
// ESTRUCTURA DE MUESTRA PARA LOS MODELOS
// ═══════════════════════════════════════════════════════

type Muestra struct {
	Features      []float64
	TargetClase   string  // Modelo 1: tipo de crimen
	TargetLat     float64 // Modelo 2: latitud
	TargetLon     float64 // Modelo 2: longitud
	TargetArresto int     // Modelo 3: 0 o 1
}

// ═══════════════════════════════════════════════════════
// NODO Y ÁRBOL DE DECISIÓN
// ═══════════════════════════════════════════════════════

type Nodo struct {
	EsHoja     bool
	ClaseHoja  string
	ValorHoja  float64
	ConteoHoja map[string]int

	FeatureIdx int
	Umbral     float64
	Izquierda  *Nodo
	Derecha    *Nodo
}

// ═══════════════════════════════════════════════════════
// CARGA DEL CSV LIMPIO
// ═══════════════════════════════════════════════════════

func dayOfWeekToInt(s string) int {
	days := map[string]int{
		"Sunday": 0, "Monday": 1, "Tuesday": 2,
		"Wednesday": 3, "Thursday": 4, "Friday": 5, "Saturday": 6,
	}
	if v, ok := days[s]; ok {
		return v
	}
	return 0
}

func CargarCSVLimpio(path string) []CrimeClean {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error abriendo CSV limpio: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.ReuseRecord = false

	// Leer encabezado
	headers, err := reader.Read()
	if err != nil {
		log.Fatalf("Error leyendo encabezado: %v", err)
	}

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
	rowNum := 0

	for {
		row, err := reader.Read()
		if err != nil {
			break
		}
		rowNum++

		// Parsear day_of_week
		dowStr := get(row, "day_of_week")
		dow := dayOfWeekToInt(dowStr)

		// victim_identified
		victIdent := get(row, "victim_identified") == "true"

		// Binarizar arresto
		statusDesc := strings.ToLower(get(row, "status_desc"))
		arresto := 0
		if strings.Contains(statusDesc, "arrest") {
			arresto = 1
		}

		c := CrimeClean{
			Hour:             toInt(get(row, "hour")),
			DayOfWeek:        dow,
			Month:            toInt(get(row, "month")),
			Year:             toInt(get(row, "year")),
			DaysToReport:     toInt(get(row, "days_to_report")),
			Area:             toInt(get(row, "area")),
			Lat:              toFloat(get(row, "lat")),
			Lon:              toFloat(get(row, "lon")),
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

		// Filtrar coordenadas inválidas
		if c.Lat == 0 && c.Lon == 0 {
			continue
		}

		registros = append(registros, c)

		if rowNum%200000 == 0 {
			fmt.Printf("  Cargados %d registros...\n", rowNum)
		}
	}

	fmt.Printf("✔ Total registros cargados: %d\n", len(registros))
	return registros
}

// ═══════════════════════════════════════════════════════
// SPLIT TRAIN / TEST
// ═══════════════════════════════════════════════════════

func SplitTrainTest(muestras []Muestra, ratio float64) ([]Muestra, []Muestra) {
	rand.Shuffle(len(muestras), func(i, j int) {
		muestras[i], muestras[j] = muestras[j], muestras[i]
	})
	splitIdx := int(float64(len(muestras)) * ratio)
	return muestras[:splitIdx], muestras[splitIdx:]
}

// ═══════════════════════════════════════════════════════
// UTILIDADES DE ÁRBOL
// ═══════════════════════════════════════════════════════

func Gini(grupos map[string]int, total int) float64 {
	if total == 0 {
		return 0
	}
	impureza := 1.0
	for _, count := range grupos {
		p := float64(count) / float64(total)
		impureza -= p * p
	}
	return impureza
}

func MSE(valores []float64) float64 {
	if len(valores) == 0 {
		return 0
	}
	sum, sumSq := 0.0, 0.0
	n := float64(len(valores))
	for _, v := range valores {
		sum += v
		sumSq += v * v
	}
	media := sum / n
	return sumSq/n - media*media
}

func Bootstrap(muestras []Muestra, rng *rand.Rand) []Muestra {
	n := len(muestras)
	resultado := make([]Muestra, n)
	for i := range resultado {
		resultado[i] = muestras[rng.Intn(n)]
	}
	return resultado
}

func ClaseMayoritaria(muestras []Muestra) string {
	conteo := make(map[string]int)
	for _, m := range muestras {
		conteo[m.TargetClase]++
	}
	mejor, maxCount := "", 0
	for clase, count := range conteo {
		if count > maxCount {
			maxCount = count
			mejor = clase
		}
	}
	return mejor
}

func MediaValores(muestras []Muestra, usarLat bool) float64 {
	if len(muestras) == 0 {
		return 0
	}
	sum := 0.0
	for _, m := range muestras {
		if usarLat {
			sum += m.TargetLat
		} else {
			sum += m.TargetLon
		}
	}
	return sum / float64(len(muestras))
}

func MejorSplitClasificacion(muestras []Muestra, featureIdxs []int) (int, float64) {
	mejorFeature, mejorUmbral, mejorGini := -1, 0.0, math.MaxFloat64

	for _, fIdx := range featureIdxs {
		vals := make([]float64, len(muestras))
		for i, m := range muestras {
			vals[i] = m.Features[fIdx]
		}
		sort.Float64s(vals)

		for i := 0; i < len(vals)-1; i++ {
			umbral := (vals[i] + vals[i+1]) / 2.0
			izqConteo := make(map[string]int)
			derConteo := make(map[string]int)
			izqTotal, derTotal := 0, 0

			for _, m := range muestras {
				if m.Features[fIdx] <= umbral {
					izqConteo[m.TargetClase]++
					izqTotal++
				} else {
					derConteo[m.TargetClase]++
					derTotal++
				}
			}

			if izqTotal == 0 || derTotal == 0 {
				continue
			}

			total := float64(izqTotal + derTotal)
			giniPond := float64(izqTotal)/total*Gini(izqConteo, izqTotal) +
				float64(derTotal)/total*Gini(derConteo, derTotal)

			if giniPond < mejorGini {
				mejorGini = giniPond
				mejorFeature = fIdx
				mejorUmbral = umbral
			}
		}
	}
	return mejorFeature, mejorUmbral
}

func MejorSplitRegresion(muestras []Muestra, featureIdxs []int, usarLat bool) (int, float64) {
	mejorFeature, mejorUmbral, mejorMSE := -1, 0.0, math.MaxFloat64

	for _, fIdx := range featureIdxs {
		vals := make([]float64, len(muestras))
		for i, m := range muestras {
			vals[i] = m.Features[fIdx]
		}
		sort.Float64s(vals)

		for i := 0; i < len(vals)-1; i++ {
			umbral := (vals[i] + vals[i+1]) / 2.0
			var izqVals, derVals []float64

			for _, m := range muestras {
				target := m.TargetLon
				if usarLat {
					target = m.TargetLat
				}
				if m.Features[fIdx] <= umbral {
					izqVals = append(izqVals, target)
				} else {
					derVals = append(derVals, target)
				}
			}

			if len(izqVals) == 0 || len(derVals) == 0 {
				continue
			}

			n := float64(len(muestras))
			msePond := float64(len(izqVals))/n*MSE(izqVals) +
				float64(len(derVals))/n*MSE(derVals)

			if msePond < mejorMSE {
				mejorMSE = msePond
				mejorFeature = fIdx
				mejorUmbral = umbral
			}
		}
	}
	return mejorFeature, mejorUmbral
}

func ConstruirArbolClasificacion(muestras []Muestra, profundidad, maxProf, minMuestras, numFeatures int, rng *rand.Rand) *Nodo {
	if len(muestras) <= minMuestras || profundidad >= maxProf {
		conteo := make(map[string]int)
		for _, m := range muestras {
			conteo[m.TargetClase]++
		}
		return &Nodo{EsHoja: true, ClaseHoja: ClaseMayoritaria(muestras), ConteoHoja: conteo}
	}

	totalFeatures := len(muestras[0].Features)
	featureIdxs := rng.Perm(totalFeatures)[:numFeatures]
	fIdx, umbral := MejorSplitClasificacion(muestras, featureIdxs)

	if fIdx == -1 {
		conteo := make(map[string]int)
		for _, m := range muestras {
			conteo[m.TargetClase]++
		}
		return &Nodo{EsHoja: true, ClaseHoja: ClaseMayoritaria(muestras), ConteoHoja: conteo}
	}

	var izq, der []Muestra
	for _, m := range muestras {
		if m.Features[fIdx] <= umbral {
			izq = append(izq, m)
		} else {
			der = append(der, m)
		}
	}

	return &Nodo{
		FeatureIdx: fIdx,
		Umbral:     umbral,
		Izquierda:  ConstruirArbolClasificacion(izq, profundidad+1, maxProf, minMuestras, numFeatures, rng),
		Derecha:    ConstruirArbolClasificacion(der, profundidad+1, maxProf, minMuestras, numFeatures, rng),
	}
}

func ConstruirArbolRegresion(muestras []Muestra, profundidad, maxProf, minMuestras, numFeatures int, usarLat bool, rng *rand.Rand) *Nodo {
	if len(muestras) <= minMuestras || profundidad >= maxProf {
		return &Nodo{EsHoja: true, ValorHoja: MediaValores(muestras, usarLat)}
	}

	totalFeatures := len(muestras[0].Features)
	featureIdxs := rng.Perm(totalFeatures)[:numFeatures]
	fIdx, umbral := MejorSplitRegresion(muestras, featureIdxs, usarLat)

	if fIdx == -1 {
		return &Nodo{EsHoja: true, ValorHoja: MediaValores(muestras, usarLat)}
	}

	var izq, der []Muestra
	for _, m := range muestras {
		if m.Features[fIdx] <= umbral {
			izq = append(izq, m)
		} else {
			der = append(der, m)
		}
	}

	return &Nodo{
		FeatureIdx: fIdx,
		Umbral:     umbral,
		Izquierda:  ConstruirArbolRegresion(izq, profundidad+1, maxProf, minMuestras, numFeatures, usarLat, rng),
		Derecha:    ConstruirArbolRegresion(der, profundidad+1, maxProf, minMuestras, numFeatures, usarLat, rng),
	}
}

func PredecirClasificacion(nodo *Nodo, features []float64) string {
	if nodo.EsHoja {
		return nodo.ClaseHoja
	}
	if features[nodo.FeatureIdx] <= nodo.Umbral {
		return PredecirClasificacion(nodo.Izquierda, features)
	}
	return PredecirClasificacion(nodo.Derecha, features)
}

func PredecirRegresion(nodo *Nodo, features []float64) float64 {
	if nodo.EsHoja {
		return nodo.ValorHoja
	}
	if features[nodo.FeatureIdx] <= nodo.Umbral {
		return PredecirRegresion(nodo.Izquierda, features)
	}
	return PredecirRegresion(nodo.Derecha, features)
}

func victimIdentifiedToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// ═══════════════════════════════════════════════════════
// ÁRBOL CON SPLIT PARALELO POR FEATURE (compartido)
// ═══════════════════════════════════════════════════════

// mejorSplitParalelo evalúa cada feature en una goroutine independiente
func mejorSplitParalelo(muestras []Muestra, featureIdxs []int) (int, float64) {
	type resultado struct {
		fIdx    int
		umbral  float64
		giniVal float64
	}

	resultCh := make(chan resultado, len(featureIdxs))
	var wg sync.WaitGroup

	for _, fIdx := range featureIdxs {
		wg.Add(1)
		go func(fi int) {
			defer wg.Done()
			mejorU, mejorG := 0.0, math.MaxFloat64

			seen := make(map[float64]bool)
			var vals []float64
			for _, m := range muestras {
				v := m.Features[fi]
				if !seen[v] {
					seen[v] = true
					vals = append(vals, v)
				}
			}

			step := 1
			if len(vals) > 20 {
				step = len(vals) / 20
			}

			for i := 0; i < len(vals)-step; i += step {
				umbral := (vals[i] + vals[i+step]) / 2.0
				izqConteo := make(map[string]int)
				derConteo := make(map[string]int)
				izqTotal, derTotal := 0, 0

				for _, m := range muestras {
					if m.Features[fi] <= umbral {
						izqConteo[m.TargetClase]++
						izqTotal++
					} else {
						derConteo[m.TargetClase]++
						derTotal++
					}
				}

				if izqTotal == 0 || derTotal == 0 {
					continue
				}

				total := float64(izqTotal + derTotal)
				giniPond := float64(izqTotal)/total*Gini(izqConteo, izqTotal) +
					float64(derTotal)/total*Gini(derConteo, derTotal)

				if giniPond < mejorG {
					mejorG = giniPond
					mejorU = umbral
				}
			}
			resultCh <- resultado{fIdx: fi, umbral: mejorU, giniVal: mejorG}
		}(fIdx)
	}

	wg.Wait()
	close(resultCh)

	mejorFeature, mejorUmbral, mejorGini := -1, 0.0, math.MaxFloat64
	for r := range resultCh {
		if r.giniVal < mejorGini {
			mejorGini = r.giniVal
			mejorFeature = r.fIdx
			mejorUmbral = r.umbral
		}
	}
	return mejorFeature, mejorUmbral
}

// construirArbolParalelo paraleliza ramas izquierda/derecha en niveles superiores
func construirArbolParalelo(muestras []Muestra, profundidad, maxProf, minMuestras, numFeatures int, rng *rand.Rand) *Nodo {
	if len(muestras) <= minMuestras || profundidad >= maxProf {
		conteo := make(map[string]int)
		for _, m := range muestras {
			conteo[m.TargetClase]++
		}
		return &Nodo{EsHoja: true, ClaseHoja: ClaseMayoritaria(muestras), ConteoHoja: conteo}
	}

	totalFeatures := len(muestras[0].Features)
	featureIdxs := rng.Perm(totalFeatures)[:numFeatures]
	fIdx, umbral := mejorSplitParalelo(muestras, featureIdxs)

	if fIdx == -1 {
		conteo := make(map[string]int)
		for _, m := range muestras {
			conteo[m.TargetClase]++
		}
		return &Nodo{EsHoja: true, ClaseHoja: ClaseMayoritaria(muestras), ConteoHoja: conteo}
	}

	var izq, der []Muestra
	for _, m := range muestras {
		if m.Features[fIdx] <= umbral {
			izq = append(izq, m)
		} else {
			der = append(der, m)
		}
	}

	nodo := &Nodo{FeatureIdx: fIdx, Umbral: umbral}

	// Paralelizar ramas solo en niveles superiores
	if profundidad < 3 && len(izq) > 500 && len(der) > 500 {
		var wg sync.WaitGroup
		wg.Add(2)
		rngIzq := rand.New(rand.NewSource(rng.Int63()))
		rngDer := rand.New(rand.NewSource(rng.Int63()))
		go func() {
			defer wg.Done()
			nodo.Izquierda = construirArbolParalelo(izq, profundidad+1, maxProf, minMuestras, numFeatures, rngIzq)
		}()
		go func() {
			defer wg.Done()
			nodo.Derecha = construirArbolParalelo(der, profundidad+1, maxProf, minMuestras, numFeatures, rngDer)
		}()
		wg.Wait()
	} else {
		nodo.Izquierda = construirArbolParalelo(izq, profundidad+1, maxProf, minMuestras, numFeatures, rng)
		nodo.Derecha = construirArbolParalelo(der, profundidad+1, maxProf, minMuestras, numFeatures, rng)
	}
	return nodo
}

// ═══════════════════════════════════════════════════════
// UTILIDADES COMPARTIDAS DE SAMPLING
// ═══════════════════════════════════════════════════════

// subsample toma n muestras aleatorias sin reemplazo del dataset
func subsample(muestras []Muestra, n int, rng *rand.Rand) []Muestra {
	if n >= len(muestras) {
		return muestras
	}
	perm := rng.Perm(len(muestras))[:n]
	resultado := make([]Muestra, n)
	for i, idx := range perm {
		resultado[i] = muestras[idx]
	}
	return resultado
}

// min retorna el menor de dos enteros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
