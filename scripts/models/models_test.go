package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func datosPrueba() []CrimeClean {
	datos := make([]CrimeClean, 120)
	for i := range datos {
		arresto := i % 2
		clase := "ROBO"
		if i%3 == 0 {
			clase = "ASALTO"
		}
		datos[i] = CrimeClean{
			Hour:             i % 24,
			DayOfWeek:        i % 7,
			Month:            i%12 + 1,
			Year:             2024,
			DaysToReport:     i % 5,
			Area:             i%21 + 1,
			Lat:              34.0 + float64(i%10)/100,
			Lon:              -118.0 - float64(i%10)/100,
			CrmCd:            200 + i%4,
			CrmCdDesc:        clase,
			Part12:           i%2 + 1,
			PremisCd:         100 + i%5,
			WeaponDesc:       []string{"NO WEAPON", "HAND GUN"}[i%2],
			VictAge:          20 + i%50,
			VictSex:          []string{"M", "F"}[i%2],
			VictDescent:      "H",
			VictimIdentified: i%2 == 0,
			Arresto:          arresto,
		}
	}
	return datos
}

func configPrueba() ConfigEntrenamiento {
	return ConfigEntrenamiento{
		NumArboles:  4,
		MaxProf:     2,
		MinMuestras: 4,
		Workers:     3,
		Seed:        7,
	}
}

func TestEntrenamientoParaleloPrediccionYMetricas(t *testing.T) {
	datos := datosPrueba()
	cfg := configPrueba()

	modelo1, err := EntrenarModelo1ConConfig(datos, cfg)
	if err != nil {
		t.Fatalf("modelo1: %v", err)
	}
	if pred := modelo1.Predecir(prepararMuestrasModelo1(datos[:1])[0].Features); pred == "" {
		t.Fatal("modelo1 no produjo predicción")
	}
	if met := modelo1.EvaluarMetricas(prepararMuestrasModelo1(datos)); math.IsNaN(met.Accuracy) {
		t.Fatal("accuracy de modelo1 es NaN")
	}

	modelo2, err := EntrenarModelo2ConConfig(datos, cfg)
	if err != nil {
		t.Fatalf("modelo2: %v", err)
	}
	lat, lon := modelo2.Predecir(prepararMuestrasModelo2(datos[:1])[0].Features)
	if lat == 0 || lon == 0 {
		t.Fatalf("modelo2 produjo coordenadas inválidas: %f, %f", lat, lon)
	}
	if met := modelo2.EvaluarMetricas(prepararMuestrasModelo2(datos)); math.IsNaN(met.RMSELatitud) {
		t.Fatal("RMSE de modelo2 es NaN")
	}

	modelo3, err := EntrenarModelo3ConConfig(datos, cfg)
	if err != nil {
		t.Fatalf("modelo3: %v", err)
	}
	_, prob := modelo3.Predecir(prepararMuestrasModelo3(datos[:1])[0].Features)
	if prob < 0 || prob > 1 {
		t.Fatalf("probabilidad inválida: %f", prob)
	}
	if met := modelo3.EvaluarMetricas(prepararMuestrasModelo3(datos)); math.IsNaN(met.F1) {
		t.Fatal("F1 de modelo3 es NaN")
	}
}

func TestPersistenciaModelo(t *testing.T) {
	datos := datosPrueba()
	cfg := configPrueba()
	modelo, err := EntrenarModelo1ConConfig(datos, cfg)
	if err != nil {
		t.Fatal(err)
	}
	artefacto := &ModeloPersistido{
		Version: versionModelo, Tipo: "model1", Algoritmo: "Random Forest implementado en Go",
		Features: featuresModelo1, Configuracion: cfg, Modelo1: modelo,
	}
	path := filepath.Join(t.TempDir(), "model1.json")
	if err := GuardarModelo(path, artefacto); err != nil {
		t.Fatal(err)
	}
	cargado, err := CargarModelo(path)
	if err != nil {
		t.Fatal(err)
	}
	if cargado.Modelo1.Predecir(prepararMuestrasModelo1(datos[:1])[0].Features) == "" {
		t.Fatal("modelo cargado no produjo predicción")
	}
}

func TestCargaCSVValidaEsquema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalido.csv")
	if err := os.WriteFile(path, []byte("hour,area\n12,1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := CargarCSVLimpioE(path)
	if err == nil || !strings.Contains(err.Error(), "faltan columnas") {
		t.Fatalf("se esperaba error de esquema, se obtuvo: %v", err)
	}
}

func TestSplitTrainTestReproducible(t *testing.T) {
	muestras := prepararMuestrasModelo1(datosPrueba())
	train1, test1, err := SplitTrainTestConSeed(muestras, 0.8, 99)
	if err != nil {
		t.Fatal(err)
	}
	train2, test2, err := SplitTrainTestConSeed(muestras, 0.8, 99)
	if err != nil {
		t.Fatal(err)
	}
	if len(train1) != 96 || len(test1) != 24 {
		t.Fatalf("split inesperado: %d/%d", len(train1), len(test1))
	}
	if train1[0].TargetClase != train2[0].TargetClase || test1[0].TargetClase != test2[0].TargetClase {
		t.Fatal("el split no es reproducible")
	}
}

func TestEntrenamientoSecuencialYParaleloSonEquivalentes(t *testing.T) {
	datos := datosPrueba()
	secuencial := configPrueba()
	secuencial.Workers = 1
	paralelo := secuencial
	paralelo.Workers = 4

	modeloSecuencial, err := EntrenarModelo1ConConfig(datos, secuencial)
	if err != nil {
		t.Fatal(err)
	}
	modeloParalelo, err := EntrenarModelo1ConConfig(datos, paralelo)
	if err != nil {
		t.Fatal(err)
	}
	for _, muestra := range prepararMuestrasModelo1(datos) {
		if modeloSecuencial.Predecir(muestra.Features) != modeloParalelo.Predecir(muestra.Features) {
			t.Fatal("modelo1 secuencial y paralelo produjeron predicciones distintas")
		}
	}

	modelo2Secuencial, err := EntrenarModelo2ConConfig(datos, secuencial)
	if err != nil {
		t.Fatal(err)
	}
	modelo2Paralelo, err := EntrenarModelo2ConConfig(datos, paralelo)
	if err != nil {
		t.Fatal(err)
	}
	for _, muestra := range prepararMuestrasModelo2(datos) {
		latSec, lonSec := modelo2Secuencial.Predecir(muestra.Features)
		latPar, lonPar := modelo2Paralelo.Predecir(muestra.Features)
		if latSec != latPar || lonSec != lonPar {
			t.Fatal("modelo2 secuencial y paralelo produjeron predicciones distintas")
		}
	}

	modelo3Secuencial, err := EntrenarModelo3ConConfig(datos, secuencial)
	if err != nil {
		t.Fatal(err)
	}
	modelo3Paralelo, err := EntrenarModelo3ConConfig(datos, paralelo)
	if err != nil {
		t.Fatal(err)
	}
	for _, muestra := range prepararMuestrasModelo3(datos) {
		claseSec, probSec := modelo3Secuencial.Predecir(muestra.Features)
		clasePar, probPar := modelo3Paralelo.Predecir(muestra.Features)
		if claseSec != clasePar || probSec != probPar {
			t.Fatal("modelo3 secuencial y paralelo produjeron predicciones distintas")
		}
	}
}
