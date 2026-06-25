package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"securitygo_pc4/cluster"
	"securitygo_pc4/db"
)

// ═══════════════════════════════════════════════════════
// SERVIDOR HTTP — SecurityGO PC4
// ═══════════════════════════════════════════════════════

type ConfigServidor struct {
	Puerto         string
	MongoURI       string
	RutaModel1     string
	RutaModel2     string
	RutaModel3     string
	WorkersPorNodo int
}

// ConfigPredeterminada usa rutas relativas desde PC4/
// Los modelos están en ../models/ (un nivel arriba)
func ConfigPredeterminada() ConfigServidor {
	return ConfigServidor{
		Puerto:         getEnv("PORT", "8080"),
		MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
		RutaModel1:     getEnv("MODEL1_PATH", "../models/model1.json"),
		RutaModel2:     getEnv("MODEL2_PATH", "../models/model2.json"),
		RutaModel3:     getEnv("MODEL3_PATH", "../models/model3.json"),
		WorkersPorNodo: 2,
	}
}

func IniciarServidor(cfg ConfigServidor) error {
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║      SecurityGO PC4 — API REST + Cluster     ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Printf("[Init] Rutas de modelos:\n")
	fmt.Printf("  Model1 → %s\n", cfg.RutaModel1)
	fmt.Printf("  Model2 → %s\n", cfg.RutaModel2)
	fmt.Printf("  Model3 → %s\n", cfg.RutaModel3)

	// 1. Conectar MongoDB
	log.Println("[Init] Conectando a MongoDB...")
	mongo, err := db.NuevoClienteMongo(cfg.MongoURI)
	if err != nil {
		return fmt.Errorf("error MongoDB: %w", err)
	}
	defer mongo.Cerrar()
	mongo.GuardarLog("INFO", "servidor", "SecurityGO PC4 iniciado")

	// 2. Iniciar cluster de workers
	log.Println("[Init] Iniciando cluster de nodos ML...")
	coord, err := cluster.NuevoCoordinador(cluster.ConfigCluster{
		RutaModel1:     cfg.RutaModel1,
		RutaModel2:     cfg.RutaModel2,
		RutaModel3:     cfg.RutaModel3,
		WorkersPorNodo: cfg.WorkersPorNodo,
	})
	if err != nil {
		return fmt.Errorf("error iniciando cluster: %w", err)
	}
	defer coord.Cerrar()

	// 3. Registrar rutas
	handler := NuevoHandler(coord, mongo)
	mux := http.NewServeMux()
	mux.HandleFunc("/predict/crime-type", handler.PredecirTipoCrimen)
	mux.HandleFunc("/predict/risk-zone", handler.PredecirZonaRiesgo)
	mux.HandleFunc("/predict/arrest-prob", handler.PredecirProbArresto)
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/predictions", handler.HistorialPredicciones)

	http.Handle("/", CORSMiddleware(LoggingMiddleware(mux)))

	// 4. Servidor con graceful shutdown
	servidor := &http.Server{
		Addr:         ":" + cfg.Puerto,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Printf("\n[Servidor] ✔ Escuchando en http://localhost:%s\n\n", cfg.Puerto)
		fmt.Println("  Endpoints disponibles:")
		fmt.Printf("  POST http://localhost:%s/predict/crime-type\n", cfg.Puerto)
		fmt.Printf("  POST http://localhost:%s/predict/risk-zone\n", cfg.Puerto)
		fmt.Printf("  POST http://localhost:%s/predict/arrest-prob\n", cfg.Puerto)
		fmt.Printf("  GET  http://localhost:%s/health\n", cfg.Puerto)
		fmt.Printf("  GET  http://localhost:%s/predictions?model=model1&limit=10\n\n", cfg.Puerto)
		if err := servidor.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Servidor] error fatal: %v\n", err)
		}
	}()

	<-quit
	log.Println("[Servidor] Señal recibida — apagando...")
	mongo.GuardarLog("INFO", "servidor", "SecurityGO PC4 detenido")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return servidor.Shutdown(ctx)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
