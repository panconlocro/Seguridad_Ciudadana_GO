package training

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════
// MODELO 2: PREDICCIÓN DE ZONA DE ALTO RIESGO
// Pregunta: ¿En qué coordenadas geográficas se concentra
// el mayor riesgo criminal dado un tipo de delito,
// hora y día específico?
// ═══════════════════════════════════════════════════════

type Modelo2 struct {
	ArbolesLat []*Nodo
	ArbolesLon []*Nodo
	NumArboles int
}

type MetricasModelo2 struct {
	MAELatitud   float64
	MAELongitud  float64
	RMSELatitud  float64
	RMSELongitud float64
	ErrorKm      float64
	Muestras     int
}

var featuresModelo2 = []string{
	"hour", "day_of_week", "month", "crm_cd", "premis_cd", "part_1_2", "area",
}

func prepararMuestrasModelo2(datos []CrimeClean) []Muestra {
	muestras := make([]Muestra, len(datos))
	for i, d := range datos {
		muestras[i] = Muestra{
			Features: []float64{
				float64(d.Hour),
				float64(d.DayOfWeek),
				float64(d.Month),
				float64(d.CrmCd),
				float64(d.PremisCd),
				float64(d.Part12),
				float64(d.Area),
			},
			TargetLat: d.Lat,
			TargetLon: d.Lon,
		}
	}
	return muestras
}

// mejorSplitRegresionParalelo evalúa cada feature en una goroutine
func mejorSplitRegresionParalelo(muestras []Muestra, featureIdxs []int, usarLat bool) (int, float64) {
	type resultado struct {
		fIdx   int
		umbral float64
		mseVal float64
	}

	resultCh := make(chan resultado, len(featureIdxs))
	var wg sync.WaitGroup

	for _, fIdx := range featureIdxs {
		wg.Add(1)
		go func(fi int) {
			defer wg.Done()
			mejorU, mejorM := 0.0, math.MaxFloat64

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
				var izqVals, derVals []float64

				for _, m := range muestras {
					target := m.TargetLon
					if usarLat {
						target = m.TargetLat
					}
					if m.Features[fi] <= umbral {
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

				if msePond < mejorM {
					mejorM = msePond
					mejorU = umbral
				}
			}
			resultCh <- resultado{fIdx: fi, umbral: mejorU, mseVal: mejorM}
		}(fIdx)
	}

	wg.Wait()
	close(resultCh)

	mejorFeature, mejorUmbral, mejorMSE := -1, 0.0, math.MaxFloat64
	for r := range resultCh {
		if r.mseVal < mejorMSE || (r.mseVal == mejorMSE && r.fIdx < mejorFeature) {
			mejorMSE = r.mseVal
			mejorFeature = r.fIdx
			mejorUmbral = r.umbral
		}
	}
	return mejorFeature, mejorUmbral
}

func construirArbolRegresionParalelo(muestras []Muestra, profundidad, maxProf, minMuestras, numFeatures int, usarLat bool, rng *rand.Rand) *Nodo {
	if len(muestras) <= minMuestras || profundidad >= maxProf {
		return &Nodo{EsHoja: true, ValorHoja: MediaValores(muestras, usarLat)}
	}

	totalFeatures := len(muestras[0].Features)
	featureIdxs := rng.Perm(totalFeatures)[:numFeatures]
	fIdx, umbral := mejorSplitRegresionParalelo(muestras, featureIdxs, usarLat)

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

	nodo := &Nodo{FeatureIdx: fIdx, Umbral: umbral}
	rngIzq := rand.New(rand.NewSource(rng.Int63()))
	rngDer := rand.New(rand.NewSource(rng.Int63()))

	if profundidad < 3 && len(izq) > 500 && len(der) > 500 {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			nodo.Izquierda = construirArbolRegresionParalelo(izq, profundidad+1, maxProf, minMuestras, numFeatures, usarLat, rngIzq)
		}()
		go func() {
			defer wg.Done()
			nodo.Derecha = construirArbolRegresionParalelo(der, profundidad+1, maxProf, minMuestras, numFeatures, usarLat, rngDer)
		}()
		wg.Wait()
	} else {
		nodo.Izquierda = construirArbolRegresionParalelo(izq, profundidad+1, maxProf, minMuestras, numFeatures, usarLat, rngIzq)
		nodo.Derecha = construirArbolRegresionParalelo(der, profundidad+1, maxProf, minMuestras, numFeatures, usarLat, rngDer)
	}
	return nodo
}

func EntrenarModelo2(datos []CrimeClean, numArboles, maxProf, minMuestras int) *Modelo2 {
	cfg := ConfigPredeterminada()
	cfg.NumArboles = numArboles
	cfg.MaxProf = maxProf
	cfg.MinMuestras = minMuestras
	modelo, _, err := EjecutarPipelineModelo2(datos, cfg, false)
	if err != nil {
		fmt.Printf("[Modelo 2] Error: %v\n", err)
		return nil
	}
	return modelo
}

func EntrenarModelo2ConConfig(datos []CrimeClean, cfg ConfigEntrenamiento) (*Modelo2, error) {
	modelo, _, err := entrenarModelo2Muestras(prepararMuestrasModelo2(datos), cfg)
	return modelo, err
}

func entrenarModelo2Muestras(train []Muestra, cfg ConfigEntrenamiento) (*Modelo2, time.Duration, error) {
	if err := cfg.Validar(); err != nil {
		return nil, 0, err
	}
	numFeatures, err := numFeaturesPara(train)
	if err != nil {
		return nil, 0, err
	}

	fmt.Printf("\n[Modelo 2] Iniciando entrenamiento...\n")
	fmt.Printf("[Modelo 2] Random Forest de regresión | Árboles por coordenada: %d | Profundidad: %d | Workers: %d\n",
		cfg.NumArboles, cfg.MaxProf, cfg.Workers)
	fmt.Printf("[Modelo 2] Muestras de entrenamiento: %d\n", len(train))

	inicio := time.Now()
	arbolesLat := make([]*Nodo, cfg.NumArboles)
	arbolesLon := make([]*Nodo, cfg.NumArboles)
	type trabajo struct {
		idx     int
		usarLat bool
	}
	type resultado struct {
		idx     int
		usarLat bool
		arbol   *Nodo
	}
	jobs := make(chan trabajo)
	results := make(chan resultado, cfg.NumArboles*2)
	var wg sync.WaitGroup
	numWorkers := min(cfg.Workers, cfg.NumArboles*2)
	for workerID := 1; workerID <= numWorkers; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Printf("[Modelo 2] Worker %d iniciado\n", id)
			for job := range jobs {
				offset := int64(5000)
				if job.usarLat {
					offset = 0
				}
				rng := rand.New(rand.NewSource(cfg.Seed + offset + int64(job.idx*31)))
				muestra := bootstrapN(train, min(maxMuestrasPorArbol, len(train)), rng)
				var arbol *Nodo
				if cfg.Workers == 1 {
					arbol = ConstruirArbolRegresion(muestra, 0, cfg.MaxProf, cfg.MinMuestras, numFeatures, job.usarLat, rng)
				} else {
					arbol = construirArbolRegresionParalelo(muestra, 0, cfg.MaxProf, cfg.MinMuestras, numFeatures, job.usarLat, rng)
				}
				results <- resultado{idx: job.idx, usarLat: job.usarLat, arbol: arbol}
			}
		}(workerID)
	}
	go func() {
		for i := 0; i < cfg.NumArboles; i++ {
			jobs <- trabajo{idx: i, usarLat: true}
			jobs <- trabajo{idx: i, usarLat: false}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	for resultado := range results {
		coord := "LON"
		if resultado.usarLat {
			coord = "LAT"
			arbolesLat[resultado.idx] = resultado.arbol
		} else {
			arbolesLon[resultado.idx] = resultado.arbol
		}
		fmt.Printf("[Modelo 2] ✔ Árbol %s %d/%d completado\n", coord, resultado.idx+1, cfg.NumArboles)
	}

	duracion := time.Since(inicio)
	fmt.Printf("[Modelo 2] ✔ Entrenamiento completado en %v con %d workers\n", duracion, numWorkers)
	return &Modelo2{ArbolesLat: arbolesLat, ArbolesLon: arbolesLon, NumArboles: cfg.NumArboles}, duracion, nil
}

func (m *Modelo2) Predecir(features []float64) (float64, float64) {
	sumLat, sumLon := 0.0, 0.0
	for i := 0; i < m.NumArboles; i++ {
		sumLat += PredecirRegresion(m.ArbolesLat[i], features)
		sumLon += PredecirRegresion(m.ArbolesLon[i], features)
	}
	return sumLat / float64(m.NumArboles), sumLon / float64(m.NumArboles)
}

func (m *Modelo2) Evaluar(test []Muestra) {
	m.EvaluarMetricas(test)
}

func (m *Modelo2) EvaluarMetricas(test []Muestra) MetricasModelo2 {
	if len(test) == 0 {
		fmt.Println("[Modelo 2] No hay muestras para evaluar")
		return MetricasModelo2{}
	}
	rng := rand.New(rand.NewSource(42))
	testMuestra := subsample(test, min(50000, len(test)), rng)

	sumErrLat, sumErrLon := 0.0, 0.0
	sumSqLat, sumSqLon := 0.0, 0.0
	n := float64(len(testMuestra))

	for _, muestra := range testMuestra {
		predLat, predLon := m.Predecir(muestra.Features)
		errLat := predLat - muestra.TargetLat
		errLon := predLon - muestra.TargetLon
		sumErrLat += math.Abs(errLat)
		sumErrLon += math.Abs(errLon)
		sumSqLat += errLat * errLat
		sumSqLon += errLon * errLon
	}

	maeLat := sumErrLat / n
	maeLon := sumErrLon / n
	rmseLat := math.Sqrt(sumSqLat / n)
	rmseLon := math.Sqrt(sumSqLon / n)
	errorKm := math.Sqrt(maeLat*maeLat+maeLon*maeLon) * 111

	fmt.Printf("[Modelo 2] ✔ MAE Latitud : %.6f grados\n", maeLat)
	fmt.Printf("[Modelo 2] ✔ MAE Longitud: %.6f grados\n", maeLon)
	fmt.Printf("[Modelo 2] ✔ RMSE Latitud : %.6f grados\n", rmseLat)
	fmt.Printf("[Modelo 2] ✔ RMSE Longitud: %.6f grados\n", rmseLon)
	fmt.Printf("[Modelo 2] ✔ Error aprox : %.2f km\n", errorKm)
	return MetricasModelo2{
		MAELatitud: maeLat, MAELongitud: maeLon,
		RMSELatitud: rmseLat, RMSELongitud: rmseLon,
		ErrorKm: errorKm, Muestras: len(testMuestra),
	}
}

func ConsultarModelo2(modelo *Modelo2) {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  CONSULTA — MODELO 2: ZONA DE RIESGO")
	fmt.Println("  Pregunta: ¿Dónde está la zona de mayor riesgo?")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	consultas := []struct {
		desc     string
		features []float64
	}{
		{"Robo vehículo (510) | Lunes | 12:00 | Área 1", []float64{12, 1, 6, 510, 101, 1, 1}},
		{"Asalto (230) | Viernes | 22:00 | Área 7", []float64{22, 5, 8, 230, 101, 1, 7}},
		{"Robo desde vehículo (330) | Sábado | 02:00 | Área 14", []float64{2, 6, 3, 330, 123, 1, 14}},
	}

	for _, c := range consultas {
		lat, lon := modelo.Predecir(c.features)
		fmt.Printf("\nConsulta : %s\n", c.desc)
		fmt.Printf("► Zona de riesgo (LAT): %.4f\n", lat)
		fmt.Printf("► Zona de riesgo (LON): %.4f\n", lon)
		fmt.Printf("► Google Maps: https://maps.google.com/?q=%.4f,%.4f\n", lat, lon)
	}
}

func EjecutarModelo2(datos []CrimeClean, numArboles, maxProf, minMuestras int) {
	cfg := ConfigPredeterminada()
	cfg.NumArboles = numArboles
	cfg.MaxProf = maxProf
	cfg.MinMuestras = minMuestras
	if _, _, err := EjecutarPipelineModelo2(datos, cfg, true); err != nil {
		fmt.Printf("[Modelo 2] Error: %v\n", err)
	}
}

func EjecutarPipelineModelo2(datos []CrimeClean, cfg ConfigEntrenamiento, consultar bool) (*Modelo2, MetricasModelo2, error) {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  MODELO 2 — PREDICCIÓN ZONA DE RIESGO")
	fmt.Println("═══════════════════════════════════════════")
	train, test, err := SplitTrainTestConSeed(prepararMuestrasModelo2(datos), 0.8, cfg.Seed)
	if err != nil {
		return nil, MetricasModelo2{}, err
	}
	fmt.Printf("[Modelo 2] Train: %d | Test: %d\n", len(train), len(test))
	modelo, _, err := entrenarModelo2Muestras(train, cfg)
	if err != nil {
		return nil, MetricasModelo2{}, err
	}
	metricas := modelo.EvaluarMetricas(test)
	if consultar {
		ConsultarModelo2(modelo)
	}
	return modelo, metricas, nil
}
