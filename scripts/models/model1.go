package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════
// MODELO 1: CLASIFICACIÓN DE TIPO DE CRIMEN
// Pregunta: ¿Qué tipo de crimen es más probable dado
// un área policial, hora del día y tipo de lugar?
// ═══════════════════════════════════════════════════════

type Modelo1 struct {
	Arboles    []*Nodo
	NumArboles int
}

type MetricasModelo1 struct {
	Accuracy float64
	Muestras int
}

var featuresModelo1 = []string{
	"hour", "day_of_week", "month", "area", "premis_cd", "part_1_2",
	"victim_identified", "days_to_report",
}

func prepararMuestrasModelo1(datos []CrimeClean) []Muestra {
	muestras := make([]Muestra, len(datos))
	for i, d := range datos {
		muestras[i] = Muestra{
			Features: []float64{
				float64(d.Hour),
				float64(d.DayOfWeek),
				float64(d.Month),
				float64(d.Area),
				float64(d.PremisCd),
				float64(d.Part12),
				victimIdentifiedToFloat(d.VictimIdentified),
				float64(d.DaysToReport),
			},
			TargetClase: d.CrmCdDesc,
		}
	}
	return muestras
}

func EntrenarModelo1(datos []CrimeClean, numArboles, maxProf, minMuestras int) *Modelo1 {
	cfg := ConfigPredeterminada()
	cfg.NumArboles = numArboles
	cfg.MaxProf = maxProf
	cfg.MinMuestras = minMuestras
	modelo, _, err := EjecutarPipelineModelo1(datos, cfg, false)
	if err != nil {
		fmt.Printf("[Modelo 1] Error: %v\n", err)
		return nil
	}
	return modelo
}

func EntrenarModelo1ConConfig(datos []CrimeClean, cfg ConfigEntrenamiento) (*Modelo1, error) {
	modelo, _, err := entrenarModelo1Muestras(prepararMuestrasModelo1(datos), cfg)
	return modelo, err
}

func entrenarModelo1Muestras(train []Muestra, cfg ConfigEntrenamiento) (*Modelo1, time.Duration, error) {
	if err := cfg.Validar(); err != nil {
		return nil, 0, err
	}
	numFeatures, err := numFeaturesPara(train)
	if err != nil {
		return nil, 0, err
	}

	fmt.Printf("\n[Modelo 1] Iniciando entrenamiento...\n")
	fmt.Printf("[Modelo 1] Random Forest | Árboles: %d | Profundidad: %d | Workers: %d\n",
		cfg.NumArboles, cfg.MaxProf, cfg.Workers)
	fmt.Printf("[Modelo 1] Muestras de entrenamiento: %d\n", len(train))

	inicio := time.Now()
	arboles := make([]*Nodo, cfg.NumArboles)
	jobs := make(chan int)
	type resultado struct {
		idx   int
		arbol *Nodo
	}
	results := make(chan resultado, cfg.NumArboles)
	var wg sync.WaitGroup
	numWorkers := min(cfg.Workers, cfg.NumArboles)
	for workerID := 1; workerID <= numWorkers; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Printf("[Modelo 1] Worker %d iniciado\n", id)
			for idx := range jobs {
				rng := rand.New(rand.NewSource(cfg.Seed + int64(idx*17)))
				muestra := bootstrapN(train, min(maxMuestrasPorArbol, len(train)), rng)
				var arbol *Nodo
				if cfg.Workers == 1 {
					arbol = ConstruirArbolClasificacion(muestra, 0, cfg.MaxProf, cfg.MinMuestras, numFeatures, rng)
				} else {
					arbol = construirArbolParalelo(muestra, 0, cfg.MaxProf, cfg.MinMuestras, numFeatures, rng)
				}
				results <- resultado{idx: idx, arbol: arbol}
			}
		}(workerID)
	}
	go func() {
		for i := 0; i < cfg.NumArboles; i++ {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	for resultado := range results {
		arboles[resultado.idx] = resultado.arbol
		fmt.Printf("[Modelo 1] ✔ Árbol %d/%d completado\n", resultado.idx+1, cfg.NumArboles)
	}

	duracion := time.Since(inicio)
	fmt.Printf("[Modelo 1] ✔ Entrenamiento completado en %v con %d workers\n", duracion, numWorkers)
	return &Modelo1{Arboles: arboles, NumArboles: cfg.NumArboles}, duracion, nil
}

func (m *Modelo1) Predecir(features []float64) string {
	votos := make(map[string]int)
	for _, arbol := range m.Arboles {
		votos[PredecirClasificacion(arbol, features)]++
	}
	mejor, maxVotos := "", 0
	for clase, v := range votos {
		if v > maxVotos || (v == maxVotos && (mejor == "" || clase < mejor)) {
			maxVotos = v
			mejor = clase
		}
	}
	return mejor
}

func (m *Modelo1) PredecirConConfianza(features []float64) (string, float64) {
	votos := make(map[string]int)
	for _, arbol := range m.Arboles {
		votos[PredecirClasificacion(arbol, features)]++
	}
	mejor, maxVotos := "", 0
	for clase, v := range votos {
		if v > maxVotos || (v == maxVotos && (mejor == "" || clase < mejor)) {
			maxVotos = v
			mejor = clase
		}
	}
	return mejor, float64(maxVotos) / float64(m.NumArboles) * 100
}

func (m *Modelo1) Evaluar(test []Muestra) float64 {
	return m.EvaluarMetricas(test).Accuracy
}

func (m *Modelo1) EvaluarMetricas(test []Muestra) MetricasModelo1 {
	if len(test) == 0 {
		fmt.Println("[Modelo 1] No hay muestras para evaluar")
		return MetricasModelo1{}
	}
	rng := rand.New(rand.NewSource(42))
	testMuestra := subsample(test, min(50000, len(test)), rng)
	correctos := 0
	for _, muestra := range testMuestra {
		if m.Predecir(muestra.Features) == muestra.TargetClase {
			correctos++
		}
	}
	acc := float64(correctos) / float64(len(testMuestra)) * 100
	fmt.Printf("[Modelo 1] ✔ Accuracy: %.2f%% sobre %d muestras de test\n", acc, len(testMuestra))
	return MetricasModelo1{Accuracy: acc, Muestras: len(testMuestra)}
}

func ConsultarModelo1(modelo *Modelo1) {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  CONSULTA — MODELO 1: TIPO DE CRIMEN")
	fmt.Println("  Pregunta: ¿Qué tipo de crimen ocurrirá?")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	consultas := []struct {
		desc     string
		features []float64
	}{
		{"Área 1 | Lunes | 12:00 | Calle | Víctima identificada", []float64{12, 1, 6, 1, 101, 1, 1, 0}},
		{"Área 7 | Viernes | 22:00 | Estacionamiento | Sin víctima", []float64{22, 5, 8, 7, 123, 1, 0, 1}},
		{"Área 14 | Sábado | 02:00 | Residencia | Víctima identificada", []float64{2, 6, 3, 14, 501, 2, 1, 0}},
	}

	for _, c := range consultas {
		pred, confianza := modelo.PredecirConConfianza(c.features)
		fmt.Printf("\nConsulta  : %s\n", c.desc)
		fmt.Printf("► Crimen más probable : %s\n", pred)
		fmt.Printf("► Confianza           : %.1f%%\n", confianza)
	}
}

func EjecutarModelo1(datos []CrimeClean, numArboles, maxProf, minMuestras int) {
	cfg := ConfigPredeterminada()
	cfg.NumArboles = numArboles
	cfg.MaxProf = maxProf
	cfg.MinMuestras = minMuestras
	if _, _, err := EjecutarPipelineModelo1(datos, cfg, true); err != nil {
		fmt.Printf("[Modelo 1] Error: %v\n", err)
	}
}

func EjecutarPipelineModelo1(datos []CrimeClean, cfg ConfigEntrenamiento, consultar bool) (*Modelo1, MetricasModelo1, error) {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  MODELO 1 — CLASIFICACIÓN TIPO DE CRIMEN")
	fmt.Println("═══════════════════════════════════════════")
	train, test, err := SplitTrainTestConSeed(prepararMuestrasModelo1(datos), 0.8, cfg.Seed)
	if err != nil {
		return nil, MetricasModelo1{}, err
	}
	fmt.Printf("[Modelo 1] Train: %d | Test: %d\n", len(train), len(test))
	modelo, _, err := entrenarModelo1Muestras(train, cfg)
	if err != nil {
		return nil, MetricasModelo1{}, err
	}
	metricas := modelo.EvaluarMetricas(test)
	if consultar {
		ConsultarModelo1(modelo)
	}
	return modelo, metricas, nil
}
