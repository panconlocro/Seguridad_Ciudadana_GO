package main

import (
	"fmt"
	"log"
	"os"

	"securitygo_pc4/api"
)

// ═══════════════════════════════════════════════════════
// SecurityGO PC4 — Punto de entrada
//
// Extensión del sistema PC3 con:
//   - Cluster distribuido de nodos ML (goroutines + channels)
//   - API REST (net/http): 3 endpoints de predicción
//   - MongoDB: persistencia de predicciones
//
// Uso:
//   go run . [--port 8080] [--mongo mongodb://localhost:27017]
//             [--model1 models/model1.json] ...
// ═══════════════════════════════════════════════════════

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	cfg := api.ConfigPredeterminada()

	// Permitir override por argumentos simples
	args := os.Args[1:]
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--port":
			cfg.Puerto = args[i+1]
		case "--mongo":
			cfg.MongoURI = args[i+1]
		case "--model1":
			cfg.RutaModel1 = args[i+1]
		case "--model2":
			cfg.RutaModel2 = args[i+1]
		case "--model3":
			cfg.RutaModel3 = args[i+1]
		case "--workers":
			fmt.Sscanf(args[i+1], "%d", &cfg.WorkersPorNodo)
		}
	}

	if err := api.IniciarServidor(cfg); err != nil {
		log.Fatalf("Error fatal: %v\n", err)
	}
}
