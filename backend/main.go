package main

import (
	"fmt"
	"log"
	"os"

	"securitygo_backend/api"
)

// ═══════════════════════════════════════════════════════
// SecurityGO PC4 — Punto de entrada
//
// Sistema distribuido con:
//   - Cluster de nodos ML con comunicación TCP
//   - API REST (net/http): endpoints de predicción
//   - MongoDB: persistencia de predicciones
//   - Redis: caché de predicciones precalculadas
//   - WebSocket: datos en tiempo real
//
// Uso:
//   go run . [--port 8080] [--mongo mongodb://localhost:27017]
//            [--redis redis://localhost:6379]
//            [--model1 models/model1.json] ...
//            [--node-port1 9001] [--node-port2 9002] [--node-port3 9003]
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
		case "--redis":
			cfg.RedisURI = args[i+1]
		case "--model1":
			cfg.RutaModel1 = args[i+1]
		case "--model2":
			cfg.RutaModel2 = args[i+1]
		case "--model3":
			cfg.RutaModel3 = args[i+1]
		case "--workers":
			fmt.Sscanf(args[i+1], "%d", &cfg.WorkersPorNodo)
		case "--node-port1":
			cfg.PuertoNodo1 = args[i+1]
		case "--node-port2":
			cfg.PuertoNodo2 = args[i+1]
		case "--node-port3":
			cfg.PuertoNodo3 = args[i+1]
		}
	}

	if err := api.IniciarServidor(cfg); err != nil {
		log.Fatalf("Error fatal: %v\n", err)
	}
}
