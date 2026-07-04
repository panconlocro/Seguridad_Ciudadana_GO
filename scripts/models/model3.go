package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════
// MODELO 3: PREDICCIÓN DE PROBABILIDAD DE ARRESTO
// Pregunta: ¿Qué probabilidad existe de que este
// incidente resulte en arresto?
// ═══════════════════════════════════════════════════════

type Modelo3 struct {
	Arboles    []*Nodo
	NumArboles int
}

type MetricasModelo3 struct {
	Accuracy  float64
	Precision float64
	Recall    float64
	F1        float64
	TP        int
	FP        int
	TN        int
	FN        int
	Muestras  int
}

var featuresModelo3 = []string{
	"crm_cd", "area", "hour", "day_of_week", "premis_cd", "weapon_present",
	"victim_identified", "days_to_report", "part_1_2",
}

func weaponToFloat(w string) float64 {
	if w == "NO WEAPON" || w == "" {
		return 0.0
	}
	return 1.0
}

func prepararMuestrasModelo3(datos []CrimeClean) []Muestra {
	muestras := make([]Muestra, len(datos))
	for i, d := range datos {
		targetClase := "NO_ARRESTO"
		if d.Arresto == 1 {
			targetClase = "ARRESTO"
		}
		muestras[i] = Muestra{
			Features: []float64{
				float64(d.CrmCd),
				float64(d.Area),
				float64(d.Hour),
				float64(d.DayOfWeek),
				float64(d.PremisCd),
				weaponToFloat(d.WeaponDesc),
				victimIdentifiedToFloat(d.VictimIdentified),
				float64(d.DaysToReport),
				float64(d.Part12),
			},
			TargetClase:   targetClase,
			TargetArresto: d.Arresto,
		}
	}
	return muestras
}

func oversampleMinoria(muestras []Muestra) ([]Muestra, error) {
	var mayoria, minoria []Muestra
	for _, m := range muestras {
		if m.TargetClase == "ARRESTO" {
			minoria = append(minoria, m)
		} else {
			mayoria = append(mayoria, m)
		}
	}
	if len(minoria) == 0 || len(mayoria) == 0 {
		return nil, fmt.Errorf("modelo 3 requiere muestras de ARRESTO y NO_ARRESTO")
	}
	factor := max(1, len(mayoria)/len(minoria))
	resultado := append([]Muestra{}, mayoria...)
	for i := 0; i < factor; i++ {
		resultado = append(resultado, minoria...)
	}
	fmt.Printf("[Modelo 3] Oversampling: %d mayoría + %d minoría x%d → %d total\n",
		len(mayoria), len(minoria), factor, len(resultado))
	return resultado, nil
}

func EntrenarModelo3(datos []CrimeClean, numArboles, maxProf, minMuestras int) *Modelo3 {
	cfg := ConfigPredeterminada()
	cfg.NumArboles = numArboles
	cfg.MaxProf = maxProf
	cfg.MinMuestras = minMuestras
	modelo, _, err := EjecutarPipelineModelo3(datos, cfg, false)
	if err != nil {
		fmt.Printf("[Modelo 3] Error: %v\n", err)
		return nil
	}
	return modelo
}

func EntrenarModelo3ConConfig(datos []CrimeClean, cfg ConfigEntrenamiento) (*Modelo3, error) {
	modelo, _, err := entrenarModelo3Muestras(prepararMuestrasModelo3(datos), cfg)
	return modelo, err
}

func entrenarModelo3Muestras(train []Muestra, cfg ConfigEntrenamiento) (*Modelo3, time.Duration, error) {
	if err := cfg.Validar(); err != nil {
		return nil, 0, err
	}
	fmt.Printf("\n[Modelo 3] Iniciando entrenamiento...\n")
	fmt.Printf("[Modelo 3] Random Forest | Árboles: %d | Profundidad: %d | Workers: %d\n",
		cfg.NumArboles, cfg.MaxProf, cfg.Workers)
	trainBalanceado, err := oversampleMinoria(train)
	if err != nil {
		return nil, 0, err
	}
	fmt.Printf("[Modelo 3] Train balanceado: %d\n", len(trainBalanceado))
	numFeatures, err := numFeaturesPara(trainBalanceado)
	if err != nil {
		return nil, 0, err
	}

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
			fmt.Printf("[Modelo 3] Worker %d iniciado\n", id)
			for idx := range jobs {
				rng := rand.New(rand.NewSource(cfg.Seed + int64(idx*23)))
				muestra := bootstrapN(trainBalanceado, min(maxMuestrasPorArbol, len(trainBalanceado)), rng)
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
		fmt.Printf("[Modelo 3] ✔ Árbol %d/%d completado\n", resultado.idx+1, cfg.NumArboles)
	}

	duracion := time.Since(inicio)
	fmt.Printf("[Modelo 3] ✔ Entrenamiento completado en %v con %d workers\n", duracion, numWorkers)
	return &Modelo3{Arboles: arboles, NumArboles: cfg.NumArboles}, duracion, nil
}

func (m *Modelo3) Predecir(features []float64) (string, float64) {
	votosArresto := 0
	for _, arbol := range m.Arboles {
		if PredecirClasificacion(arbol, features) == "ARRESTO" {
			votosArresto++
		}
	}
	prob := float64(votosArresto) / float64(m.NumArboles)
	if prob >= 0.5 {
		return "ARRESTO", prob
	}
	return "NO_ARRESTO", prob
}

func (m *Modelo3) Evaluar(test []Muestra) {
	m.EvaluarMetricas(test)
}

func (m *Modelo3) EvaluarMetricas(test []Muestra) MetricasModelo3 {
	if len(test) == 0 {
		fmt.Println("[Modelo 3] No hay muestras para evaluar")
		return MetricasModelo3{}
	}
	rng := rand.New(rand.NewSource(42))
	testMuestra := subsample(test, min(50000, len(test)), rng)

	tp, fp, tn, fn := 0, 0, 0, 0
	for _, muestra := range testMuestra {
		pred, _ := m.Predecir(muestra.Features)
		real := muestra.TargetClase
		switch {
		case pred == "ARRESTO" && real == "ARRESTO":
			tp++
		case pred == "ARRESTO" && real == "NO_ARRESTO":
			fp++
		case pred == "NO_ARRESTO" && real == "NO_ARRESTO":
			tn++
		case pred == "NO_ARRESTO" && real == "ARRESTO":
			fn++
		}
	}

	total := float64(len(testMuestra))
	accuracy := float64(tp+tn) / total * 100
	var precision, recall, f1 float64
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp) * 100
	}
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn) * 100
	}
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}

	fmt.Printf("[Modelo 3] ✔ Accuracy  : %.2f%%\n", accuracy)
	fmt.Printf("[Modelo 3] ✔ Precision : %.2f%%\n", precision)
	fmt.Printf("[Modelo 3] ✔ Recall    : %.2f%%\n", recall)
	fmt.Printf("[Modelo 3] ✔ F1-score  : %.2f%%\n", f1)
	fmt.Printf("[Modelo 3]   TP:%d FP:%d TN:%d FN:%d\n", tp, fp, tn, fn)
	return MetricasModelo3{
		Accuracy: accuracy, Precision: precision, Recall: recall, F1: f1,
		TP: tp, FP: fp, TN: tn, FN: fn, Muestras: len(testMuestra),
	}
}

func ConsultarModelo3(modelo *Modelo3) {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  CONSULTA — MODELO 3: PROBABILIDAD DE ARRESTO")
	fmt.Println("  Pregunta: ¿Habrá arresto en este incidente?")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	consultas := []struct {
		desc     string
		features []float64
	}{
		{
			"Robo vehículo | Área 1 | 12:00 | Sin arma | Víctima presente",
			[]float64{510, 1, 12, 1, 101, 0, 1, 0, 1},
		},
		{
			"Asalto | Área 7 | 22:00 | Con arma | Víctima presente",
			[]float64{230, 7, 22, 5, 101, 1, 1, 0, 1},
		},
		{
			"Robo desde vehículo | Área 14 | 02:00 | Sin víctima | Reporte tardío 5 días",
			[]float64{330, 14, 2, 6, 123, 0, 0, 5, 1},
		},
	}

	for _, c := range consultas {
		resultado, prob := modelo.Predecir(c.features)
		fmt.Printf("\nConsulta : %s\n", c.desc)
		fmt.Printf("► Predicción         : %s\n", resultado)
		fmt.Printf("► Prob. de arresto   : %.1f%%\n", prob*100)
		if prob >= 0.5 {
			fmt.Printf("► Interpretación     : Alta prob. de arresto — priorizar recursos aquí\n")
		} else {
			fmt.Printf("► Interpretación     : Baja prob. de arresto — reforzar presencia policial\n")
		}
	}
}

func EjecutarModelo3(datos []CrimeClean, numArboles, maxProf, minMuestras int) {
	cfg := ConfigPredeterminada()
	cfg.NumArboles = numArboles
	cfg.MaxProf = maxProf
	cfg.MinMuestras = minMuestras
	if _, _, err := EjecutarPipelineModelo3(datos, cfg, true); err != nil {
		fmt.Printf("[Modelo 3] Error: %v\n", err)
	}
}

func EjecutarPipelineModelo3(datos []CrimeClean, cfg ConfigEntrenamiento, consultar bool) (*Modelo3, MetricasModelo3, error) {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  MODELO 3 — PROBABILIDAD DE ARRESTO")
	fmt.Println("═══════════════════════════════════════════")
	train, test, err := SplitTrainTestConSeed(prepararMuestrasModelo3(datos), 0.8, cfg.Seed)
	if err != nil {
		return nil, MetricasModelo3{}, err
	}
	fmt.Printf("[Modelo 3] Train: %d | Test: %d\n", len(train), len(test))
	modelo, _, err := entrenarModelo3Muestras(train, cfg)
	if err != nil {
		return nil, MetricasModelo3{}, err
	}
	metricas := modelo.EvaluarMetricas(test)
	if consultar {
		ConsultarModelo3(modelo)
	}
	return modelo, metricas, nil
}
