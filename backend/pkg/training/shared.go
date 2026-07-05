package training

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var columnasRequeridas = []string{
	"hour",
	"day_of_week",
	"month",
	"year",
	"days_to_report",
	"area",
	"lat",
	"lon",
	"crm_cd",
	"crm_cd_desc",
	"part_1_2",
	"premis_cd",
	"weapon_desc",
	"vict_age",
	"vict_sex",
	"vict_descent",
	"victim_identified",
	"status_desc",
}

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
	registros, err := CargarCSVLimpioE(path)
	if err != nil {
		log.Fatalf("Error cargando CSV limpio: %v", err)
	}
	return registros
}

func CargarCSVLimpioE(path string) ([]CrimeClean, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir %q: %w", path, err)
	}
	defer file.Close()
	return cargarCSVDesdeReader(file)
}

func CargarCSVLimpioDesdeBytesE(data []byte) ([]CrimeClean, error) {
	return cargarCSVDesdeReader(bytes.NewReader(data))
}

func cargarCSVDesdeReader(r io.Reader) ([]CrimeClean, error) {
	reader := csv.NewReader(r)
	reader.ReuseRecord = false

	// Leer encabezado
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el encabezado: %w", err)
	}

	idx := make(map[string]int)
	for i, h := range headers {
		idx[h] = i
	}
	var faltantes []string
	for _, columna := range columnasRequeridas {
		if _, ok := idx[columna]; !ok {
			faltantes = append(faltantes, columna)
		}
	}
	if len(faltantes) > 0 {
		return nil, fmt.Errorf("CSV inválido: faltan columnas requeridas: %s", strings.Join(faltantes, ", "))
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
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error leyendo fila %d: %w", rowNum+2, err)
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

	if len(registros) < 2 {
		return nil, fmt.Errorf("CSV inválido: se requieren al menos 2 registros válidos y se encontraron %d", len(registros))
	}
	fmt.Printf("✔ Total registros cargados: %d\n", len(registros))
	return registros, nil
}

// ═══════════════════════════════════════════════════════
// SPLIT TRAIN / TEST
// ═══════════════════════════════════════════════════════

func SplitTrainTest(muestras []Muestra, ratio float64) ([]Muestra, []Muestra) {
	train, test, err := SplitTrainTestConSeed(muestras, ratio, 42)
	if err != nil {
		return nil, nil
	}
	return train, test
}

func SplitTrainTestConSeed(muestras []Muestra, ratio float64, seed int64) ([]Muestra, []Muestra, error) {
	if len(muestras) < 2 {
		return nil, nil, fmt.Errorf("se requieren al menos 2 muestras para separar train/test")
	}
	if ratio <= 0 || ratio >= 1 {
		return nil, nil, fmt.Errorf("el ratio train/test debe estar entre 0 y 1")
	}

	barajadas := append([]Muestra(nil), muestras...)
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(barajadas), func(i, j int) {
		barajadas[i], barajadas[j] = barajadas[j], barajadas[i]
	})
	splitIdx := int(float64(len(barajadas)) * ratio)
	if splitIdx < 1 {
		splitIdx = 1
	}
	if splitIdx >= len(barajadas) {
		splitIdx = len(barajadas) - 1
	}
	return barajadas[:splitIdx], barajadas[splitIdx:], nil
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
	return bootstrapN(muestras, len(muestras), rng)
}

func bootstrapN(muestras []Muestra, n int, rng *rand.Rand) []Muestra {
	if n > len(muestras) {
		n = len(muestras)
	}
	if n < 1 || len(muestras) == 0 {
		return nil
	}
	resultado := make([]Muestra, n)
	for i := range resultado {
		resultado[i] = muestras[rng.Intn(len(muestras))]
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
		if count > maxCount || (count == maxCount && (mejor == "" || clase < mejor)) {
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
		seen := make(map[float64]bool)
		var vals []float64
		for _, m := range muestras {
			valor := m.Features[fIdx]
			if !seen[valor] {
				seen[valor] = true
				vals = append(vals, valor)
			}
		}
		sort.Float64s(vals)

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

			if giniPond < mejorGini || (giniPond == mejorGini && fIdx < mejorFeature) {
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
		seen := make(map[float64]bool)
		var vals []float64
		for _, m := range muestras {
			valor := m.Features[fIdx]
			if !seen[valor] {
				seen[valor] = true
				vals = append(vals, valor)
			}
		}
		sort.Float64s(vals)

		step := 1
		if len(vals) > 20 {
			step = len(vals) / 20
		}
		for i := 0; i < len(vals)-step; i += step {
			umbral := (vals[i] + vals[i+step]) / 2.0
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

			if msePond < mejorMSE || (msePond == mejorMSE && fIdx < mejorFeature) {
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

	rngIzq := rand.New(rand.NewSource(rng.Int63()))
	rngDer := rand.New(rand.NewSource(rng.Int63()))
	return &Nodo{
		FeatureIdx: fIdx,
		Umbral:     umbral,
		Izquierda:  ConstruirArbolClasificacion(izq, profundidad+1, maxProf, minMuestras, numFeatures, rngIzq),
		Derecha:    ConstruirArbolClasificacion(der, profundidad+1, maxProf, minMuestras, numFeatures, rngDer),
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

	rngIzq := rand.New(rand.NewSource(rng.Int63()))
	rngDer := rand.New(rand.NewSource(rng.Int63()))
	return &Nodo{
		FeatureIdx: fIdx,
		Umbral:     umbral,
		Izquierda:  ConstruirArbolRegresion(izq, profundidad+1, maxProf, minMuestras, numFeatures, usarLat, rngIzq),
		Derecha:    ConstruirArbolRegresion(der, profundidad+1, maxProf, minMuestras, numFeatures, usarLat, rngDer),
	}
}

func PredecirClasificacion(nodo *Nodo, features []float64) string {
	if nodo == nil {
		return ""
	}
	if nodo.EsHoja {
		return nodo.ClaseHoja
	}
	if nodo.FeatureIdx < 0 || nodo.FeatureIdx >= len(features) {
		return ""
	}
	if features[nodo.FeatureIdx] <= nodo.Umbral {
		return PredecirClasificacion(nodo.Izquierda, features)
	}
	return PredecirClasificacion(nodo.Derecha, features)
}

func PredecirRegresion(nodo *Nodo, features []float64) float64 {
	if nodo == nil {
		return 0
	}
	if nodo.EsHoja {
		return nodo.ValorHoja
	}
	if nodo.FeatureIdx < 0 || nodo.FeatureIdx >= len(features) {
		return 0
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
			sort.Float64s(vals)

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
		if r.giniVal < mejorGini || (r.giniVal == mejorGini && r.fIdx < mejorFeature) {
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
	rngIzq := rand.New(rand.NewSource(rng.Int63()))
	rngDer := rand.New(rand.NewSource(rng.Int63()))

	// Paralelizar ramas solo en niveles superiores
	if profundidad < 3 && len(izq) > 500 && len(der) > 500 {
		var wg sync.WaitGroup
		wg.Add(2)
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
		nodo.Izquierda = construirArbolParalelo(izq, profundidad+1, maxProf, minMuestras, numFeatures, rngIzq)
		nodo.Derecha = construirArbolParalelo(der, profundidad+1, maxProf, minMuestras, numFeatures, rngDer)
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
