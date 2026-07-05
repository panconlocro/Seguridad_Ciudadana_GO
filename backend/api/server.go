package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"securitygo_backend/cluster"
	"securitygo_backend/db"

	"golang.org/x/crypto/bcrypt"
)

// ═══════════════════════════════════════════════════════
// SERVIDOR HTTP — SecurityGO PC4
// ═══════════════════════════════════════════════════════

type ConfigServidor struct {
	Puerto         string
	MongoURI       string
	RedisURI       string
	RutaModel1     string
	RutaModel2     string
	RutaModel3     string
	FrontendDist   string // Ruta para archivos estáticos del frontend
	WorkersPorNodo int
	PuertoNodo1    string // Puerto TCP para nodo model1
	PuertoNodo2    string // Puerto TCP para nodo model2
	PuertoNodo3    string // Puerto TCP para nodo model3
}

// ConfigPredeterminada usa rutas relativas al backend o configurables
func ConfigPredeterminada() ConfigServidor {
	return ConfigServidor{
		Puerto:         getEnv("PORT", "8080"),
		MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
		RedisURI:       getEnv("REDIS_URI", "redis://localhost:6379"),
		RutaModel1:     getEnv("MODEL1_PATH", "./models_cache/model1.json"),
		RutaModel2:     getEnv("MODEL2_PATH", "./models_cache/model2.json"),
		RutaModel3:     getEnv("MODEL3_PATH", "./models_cache/model3.json"),
		FrontendDist:   getEnv("FRONTEND_DIST", "./public"),
		WorkersPorNodo: 2,
		PuertoNodo1:    getEnv("NODE1_PORT", "9001"),
		PuertoNodo2:    getEnv("NODE2_PORT", "9002"),
		PuertoNodo3:    getEnv("NODE3_PORT", "9003"),
	}
}

func IniciarServidor(cfg ConfigServidor) error {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║   SecurityGO PC4 — API + Cluster TCP + Redis    ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("[Config] Modelos:\n")
	fmt.Printf("  Model1 → %s\n", cfg.RutaModel1)
	fmt.Printf("  Model2 → %s\n", cfg.RutaModel2)
	fmt.Printf("  Model3 → %s\n", cfg.RutaModel3)
	fmt.Printf("[Config] Nodos TCP:\n")
	fmt.Printf("  model1 → :%s\n", cfg.PuertoNodo1)
	fmt.Printf("  model2 → :%s\n", cfg.PuertoNodo2)
	fmt.Printf("  model3 → :%s\n", cfg.PuertoNodo3)
	fmt.Println()

	// ── 1. Conectar MongoDB ──
	log.Println("[Init] Conectando a MongoDB...")
	mongo, err := db.NuevoClienteMongo(cfg.MongoURI)
	if err != nil {
		return fmt.Errorf("error MongoDB: %w", err)
	}
	defer mongo.Cerrar()
	mongo.GuardarLog("INFO", "servidor", "Modelos Iniciados")

	// Inicializar usuario admin por defecto si no hay ninguno
	hashAdmin, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	mongo.InicializarAdmin(string(hashAdmin))

	// ── 2. Conectar Redis (modo degradado si no disponible) ──
	log.Println("[Init] Conectando a Redis...")
	var redisClient *db.ClienteRedis
	redisClient, err = db.NuevoClienteRedis(cfg.RedisURI)
	if err != nil {
		log.Printf("[Init] ⚠ Redis no disponible: %v — continuando sin caché\n", err)
		redisClient = nil
	}
	if redisClient != nil {
		defer redisClient.Cerrar()
	}

	// ── 3. Iniciar nodos TCP (cada uno carga su modelo ML) ──
	log.Println("[Init] Iniciando nodos TCP del cluster...")

	// Crear directorio de caché de modelos si no existe
	if err := os.MkdirAll(filepath.Dir(cfg.RutaModel1), 0755); err != nil {
		log.Printf("[Init] Advertencia: no se pudo crear directorio de caché de modelos: %v", err)
	}

	type cfgNodo struct {
		id, puerto, ruta, modelo string
	}
	nodosConfig := []cfgNodo{
		{"nodo-model1", cfg.PuertoNodo1, cfg.RutaModel1, "model1"},
		{"nodo-model2", cfg.PuertoNodo2, cfg.RutaModel2, "model2"},
		{"nodo-model3", cfg.PuertoNodo3, cfg.RutaModel3, "model3"},
	}

	var nodosTCP []*cluster.NodoTCP
	for _, n := range nodosConfig {
		// Intentar descargar modelo actualizado desde GridFS
		nombreGridFS := n.modelo + ".json"
		if mongo.ExisteArchivoGridFS(nombreGridFS) {
			data, err := mongo.ObtenerArchivoGridFS(nombreGridFS)
			if err == nil {
				log.Printf("[Init] Modelo %s encontrado en GridFS, usando versión actualizada", n.modelo)
				os.WriteFile(n.ruta, data, 0644)
			} else {
				log.Printf("[Init] Error descargando modelo %s de GridFS: %v. Usando local.", n.modelo, err)
			}
		}

		nodo, err := cluster.NuevoNodoTCP(n.id, n.puerto, n.ruta, n.modelo)
		if err != nil {
			return fmt.Errorf("error creando nodo TCP %s: %w", n.id, err)
		}
		if err := nodo.Iniciar(); err != nil {
			return fmt.Errorf("error iniciando nodo TCP %s: %w", n.id, err)
		}
		nodosTCP = append(nodosTCP, nodo)
	}
	defer func() {
		for _, n := range nodosTCP {
			n.Cerrar()
		}
	}()

	// ── 4. Crear coordinador TCP (conecta a los nodos) ──
	log.Println("[Init] Creando coordinador TCP...")
	coord, err := cluster.NuevoCoordinador(cluster.ConfigCluster{
		Nodos: []cluster.ConfigNodo{
			{Modelo: "model1", Direccion: "localhost:" + cfg.PuertoNodo1},
			{Modelo: "model2", Direccion: "localhost:" + cfg.PuertoNodo2},
			{Modelo: "model3", Direccion: "localhost:" + cfg.PuertoNodo3},
		},
	})
	if err != nil {
		return fmt.Errorf("error iniciando coordinador: %w", err)
	}
	defer coord.Cerrar()

	// ── 5. Crear hub WebSocket ──
	hub := NuevoHubWS(coord)

	// ── 6. Registrar rutas ──
	handler := NuevoHandler(coord, mongo, redisClient, hub, nodosTCP)
	mux := http.NewServeMux()

	// Rutas públicas (sin JWT)
	mux.HandleFunc("/login", handler.Login)
	mux.HandleFunc("/register", handler.Register)
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/ws", hub.HandleWS)

	// Rutas protegidas con JWT
	protegido := http.NewServeMux()
	protegido.HandleFunc("/predict/crime-type", handler.PredecirTipoCrimen)
	protegido.HandleFunc("/predict/risk-zone", handler.PredecirZonaRiesgo)
	protegido.HandleFunc("/predict/arrest-prob", handler.PredecirProbArresto)
	protegido.HandleFunc("/predictions", handler.HistorialPredicciones)
	protegido.HandleFunc("/cache/stats", handler.EstadisticasCache)
	protegido.HandleFunc("/train", handler.EntrenarModelo)
	mux.Handle("/predict/", JWTMiddleware(protegido))
	mux.Handle("/predictions", JWTMiddleware(protegido))
	mux.Handle("/cache/", JWTMiddleware(protegido))
	mux.Handle("/train", JWTMiddleware(protegido))

	// Servir frontend estático (SPA)
	fs := http.FileServer(http.Dir(cfg.FrontendDist))
	mux.Handle("/app/", http.StripPrefix("/app/", fs))

	http.Handle("/", CORSMiddleware(LoggingMiddleware(mux)))

	// ── 7. Servidor con graceful shutdown ──
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
		fmt.Printf("  POST http://localhost:%s/login\n", cfg.Puerto)
		fmt.Printf("  POST http://localhost:%s/register\n", cfg.Puerto)
		fmt.Printf("  POST http://localhost:%s/predict/crime-type\n", cfg.Puerto)
		fmt.Printf("  POST http://localhost:%s/predict/risk-zone\n", cfg.Puerto)
		fmt.Printf("  POST http://localhost:%s/predict/arrest-prob\n", cfg.Puerto)
		fmt.Printf("  GET  http://localhost:%s/health\n", cfg.Puerto)
		fmt.Printf("  GET  http://localhost:%s/predictions?model=model1&limit=10\n", cfg.Puerto)
		fmt.Printf("  GET  http://localhost:%s/cache/stats\n", cfg.Puerto)
		fmt.Printf("  WS   ws://localhost:%s/ws\n\n", cfg.Puerto)
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
