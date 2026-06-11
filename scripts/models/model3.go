package main

import (
	"fmt"
	"math"
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

func oversampleMinoria(muestras []Muestra) []Muestra {
	var mayoria, minoria []Muestra
	for _, m := range muestras {
		if m.TargetClase == "ARRESTO" {
			minoria = append(minoria, m)
		} else {
			mayoria = append(mayoria, m)
		}
	}
	factor := len(mayoria) / len(minoria)
	resultado := append([]Muestra{}, mayoria...)
	for i := 0; i < factor; i++ {
		resultado = append(resultado, minoria...)
	}
	fmt.Printf("[Modelo 3] Oversampling: %d mayoría + %d minoría x%d → %d total\n",
		len(mayoria), len(minoria), factor, len(resultado))
	return resultado
}

func EntrenarModelo3(datos []CrimeClean, numArboles, maxProf, minMuestras int) *Modelo3 {
	fmt.Printf("\n[Modelo 3] Iniciando entrenamiento...\n")
	fmt.Printf("[Modelo 3] Árboles: %d | Profundidad máx: %d | Mín muestras: %d\n",
		numArboles, maxProf, minMuestras)

	todasMuestras := prepararMuestrasModelo3(datos)
	train, test := SplitTrainTest(todasMuestras, 0.8)
	fmt.Printf("[Modelo 3] Train original: %d | Test: %d\n", len(train), len(test))

	trainBalanceado := oversampleMinoria(train)
	fmt.Printf("[Modelo 3] Train balanceado: %d\n", len(trainBalanceado))

	numFeatures := int(math.Sqrt(float64(len(trainBalanceado[0].Features))))
	arboles := make([]*Nodo, numArboles)

	inicio := time.Now()
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for i := 0; i < numArboles; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(idx*23)))
			muestra := subsample(trainBalanceado, 100000, rng)
			arboles[idx] = construirArbolParalelo(muestra, 0, maxProf, minMuestras, numFeatures, rng)
			fmt.Printf("[Modelo 3] ✔ Árbol %d/%d completado\n", idx+1, numArboles)
		}(i)
	}
	wg.Wait()

	fmt.Printf("[Modelo 3] ✔ Entrenamiento completado en %v\n", time.Since(inicio))
	modelo := &Modelo3{Arboles: arboles, NumArboles: numArboles}
	modelo.Evaluar(test)
	return modelo
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

	total := float64(tp + fp + tn + fn)
	accuracy := float64(tp+tn) / total * 100
	var precision, recall float64
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp) * 100
	}
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn) * 100
	}

	fmt.Printf("[Modelo 3] ✔ Accuracy  : %.2f%%\n", accuracy)
	fmt.Printf("[Modelo 3] ✔ Precision : %.2f%%\n", precision)
	fmt.Printf("[Modelo 3] ✔ Recall    : %.2f%%\n", recall)
	fmt.Printf("[Modelo 3]   TP:%d FP:%d TN:%d FN:%d\n", tp, fp, tn, fn)
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

func main() {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  MODELO 3 — PROBABILIDAD DE ARRESTO")
	fmt.Println("═══════════════════════════════════════════")
	datos := CargarCSVLimpio("../../data/processed/Crime_Data_Clean.csv")
	modelo := EntrenarModelo3(datos, 10, 8, 50)
	ConsultarModelo3(modelo)
}

// Evitar warning de import no usado
var _ = math.Sqrt
