package main

import (
	"fmt"
	"math"
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
	fmt.Printf("\n[Modelo 1] Iniciando entrenamiento...\n")
	fmt.Printf("[Modelo 1] Árboles: %d | Profundidad máx: %d | Mín muestras: %d\n",
		numArboles, maxProf, minMuestras)

	todasMuestras := prepararMuestrasModelo1(datos)
	train, test := SplitTrainTest(todasMuestras, 0.8)
	fmt.Printf("[Modelo 1] Train: %d | Test: %d\n", len(train), len(test))

	numFeatures := int(math.Sqrt(float64(len(train[0].Features))))
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
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(idx*17)))
			muestra := subsample(train, 100000, rng)
			arboles[idx] = construirArbolParalelo(muestra, 0, maxProf, minMuestras, numFeatures, rng)
			fmt.Printf("[Modelo 1] ✔ Árbol %d/%d completado\n", idx+1, numArboles)
		}(i)
	}
	wg.Wait()

	fmt.Printf("[Modelo 1] ✔ Entrenamiento completado en %v\n", time.Since(inicio))
	modelo := &Modelo1{Arboles: arboles, NumArboles: numArboles}
	modelo.Evaluar(test)
	return modelo
}

func (m *Modelo1) Predecir(features []float64) string {
	votos := make(map[string]int)
	for _, arbol := range m.Arboles {
		votos[PredecirClasificacion(arbol, features)]++
	}
	mejor, maxVotos := "", 0
	for clase, v := range votos {
		if v > maxVotos {
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
		if v > maxVotos {
			maxVotos = v
			mejor = clase
		}
	}
	return mejor, float64(maxVotos) / float64(m.NumArboles) * 100
}

func (m *Modelo1) Evaluar(test []Muestra) float64 {
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
	return acc
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
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  MODELO 1 — CLASIFICACIÓN TIPO DE CRIMEN")
	fmt.Println("═══════════════════════════════════════════")
	modelo := EntrenarModelo1(datos, numArboles, maxProf, minMuestras)
	ConsultarModelo1(modelo)
}
