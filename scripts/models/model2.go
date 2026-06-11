package main

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
		if r.mseVal < mejorMSE {
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

	if profundidad < 3 && len(izq) > 500 && len(der) > 500 {
		var wg sync.WaitGroup
		wg.Add(2)
		rngIzq := rand.New(rand.NewSource(rng.Int63()))
		rngDer := rand.New(rand.NewSource(rng.Int63()))
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
		nodo.Izquierda = construirArbolRegresionParalelo(izq, profundidad+1, maxProf, minMuestras, numFeatures, usarLat, rng)
		nodo.Derecha = construirArbolRegresionParalelo(der, profundidad+1, maxProf, minMuestras, numFeatures, usarLat, rng)
	}
	return nodo
}

func EntrenarModelo2(datos []CrimeClean, numArboles, maxProf, minMuestras int) *Modelo2 {
	fmt.Printf("\n[Modelo 2] Iniciando entrenamiento...\n")
	fmt.Printf("[Modelo 2] Árboles: %d | Profundidad máx: %d | Mín muestras: %d\n",
		numArboles, maxProf, minMuestras)

	todasMuestras := prepararMuestrasModelo2(datos)
	train, test := SplitTrainTest(todasMuestras, 0.8)
	fmt.Printf("[Modelo 2] Train: %d | Test: %d\n", len(train), len(test))

	numFeatures := int(math.Sqrt(float64(len(train[0].Features))))
	arbolesLat := make([]*Nodo, numArboles)
	arbolesLon := make([]*Nodo, numArboles)

	inicio := time.Now()
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for i := 0; i < numArboles; i++ {
		wg.Add(2)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(idx*13)))
			muestra := subsample(train, 100000, rng)
			arbolesLat[idx] = construirArbolRegresionParalelo(muestra, 0, maxProf, minMuestras, numFeatures, true, rng)
			fmt.Printf("[Modelo 2] ✔ Árbol LAT %d/%d completado\n", idx+1, numArboles)
		}(i)

		go func(idx int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(idx*31+5000)))
			muestra := subsample(train, 100000, rng)
			arbolesLon[idx] = construirArbolRegresionParalelo(muestra, 0, maxProf, minMuestras, numFeatures, false, rng)
			fmt.Printf("[Modelo 2] ✔ Árbol LON %d/%d completado\n", idx+1, numArboles)
		}(i)
	}
	wg.Wait()

	fmt.Printf("[Modelo 2] ✔ Entrenamiento completado en %v\n", time.Since(inicio))
	modelo := &Modelo2{ArbolesLat: arbolesLat, ArbolesLon: arbolesLon, NumArboles: numArboles}
	modelo.Evaluar(test)
	return modelo
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
	rng := rand.New(rand.NewSource(42))
	testMuestra := subsample(test, min(50000, len(test)), rng)

	sumErrLat, sumErrLon := 0.0, 0.0
	n := float64(len(testMuestra))

	for _, muestra := range testMuestra {
		predLat, predLon := m.Predecir(muestra.Features)
		sumErrLat += math.Abs(predLat - muestra.TargetLat)
		sumErrLon += math.Abs(predLon - muestra.TargetLon)
	}

	maeLat := sumErrLat / n
	maeLon := sumErrLon / n
	errorKm := math.Sqrt(maeLat*maeLat+maeLon*maeLon) * 111

	fmt.Printf("[Modelo 2] ✔ MAE Latitud : %.6f grados\n", maeLat)
	fmt.Printf("[Modelo 2] ✔ MAE Longitud: %.6f grados\n", maeLon)
	fmt.Printf("[Modelo 2] ✔ Error aprox : %.2f km\n", errorKm)
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

func main() {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  MODELO 2 — PREDICCIÓN ZONA DE RIESGO")
	fmt.Println("═══════════════════════════════════════════")
	datos := CargarCSVLimpio("../../data/processed/Crime_Data_Clean.csv")
	modelo := EntrenarModelo2(datos, 10, 8, 50)
	ConsultarModelo2(modelo)
}
