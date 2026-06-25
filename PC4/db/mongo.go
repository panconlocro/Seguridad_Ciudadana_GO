package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ═══════════════════════════════════════════════════════
// CAPA DE BASE DE DATOS — MongoDB
// Guarda cada predicción realizada por el cluster
// ═══════════════════════════════════════════════════════

const (
	NombreDB        = "securitygo"
	ColeccionPred   = "predictions"
	ColeccionLogs   = "cluster_logs"
	TimeoutConexion = 10 * time.Second
)

// RegistroPrediccion representa un documento en MongoDB
type RegistroPrediccion struct {
	ID         string                 `bson:"_id,omitempty"`
	Timestamp  time.Time              `bson:"timestamp"`
	Modelo     string                 `bson:"modelo"`      // "model1", "model2", "model3"
	NodoWorker string                 `bson:"nodo_worker"` // ID del nodo que respondió
	Features   map[string]interface{} `bson:"features"`    // inputs del usuario
	Resultado  map[string]interface{} `bson:"resultado"`   // predicción devuelta
	DuracionMs int64                  `bson:"duracion_ms"` // tiempo de inferencia
}

// RegistroLog representa un evento del cluster
type RegistroLog struct {
	Timestamp time.Time `bson:"timestamp"`
	Nivel     string    `bson:"nivel"` // "INFO", "ERROR", "WARN"
	Nodo      string    `bson:"nodo"`
	Mensaje   string    `bson:"mensaje"`
}

// ClienteMongo encapsula la conexión y operaciones
type ClienteMongo struct {
	cliente *mongo.Client
	db      *mongo.Database
}

// NuevoClienteMongo crea y verifica la conexión a MongoDB
func NuevoClienteMongo(uri string) (*ClienteMongo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), TimeoutConexion)
	defer cancel()

	opts := options.Client().ApplyURI(uri)
	cliente, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("[MongoDB] error conectando: %w", err)
	}

	// Verificar conexión con ping
	if err := cliente.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("[MongoDB] ping fallido: %w", err)
	}

	log.Printf("[MongoDB] ✔ Conectado a %s — base de datos: %s\n", uri, NombreDB)
	return &ClienteMongo{
		cliente: cliente,
		db:      cliente.Database(NombreDB),
	}, nil
}

// Cerrar cierra la conexión con MongoDB
func (c *ClienteMongo) Cerrar() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.cliente.Disconnect(ctx); err != nil {
		log.Printf("[MongoDB] error al cerrar: %v\n", err)
	} else {
		log.Println("[MongoDB] Conexión cerrada correctamente")
	}
}

// GuardarPrediccion inserta un registro de predicción en MongoDB
func (c *ClienteMongo) GuardarPrediccion(reg RegistroPrediccion) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reg.Timestamp = time.Now().UTC()
	col := c.db.Collection(ColeccionPred)
	res, err := col.InsertOne(ctx, reg)
	if err != nil {
		return fmt.Errorf("[MongoDB] error guardando predicción: %w", err)
	}
	log.Printf("[MongoDB] ✔ Predicción guardada — ID: %v | Modelo: %s | Nodo: %s\n",
		res.InsertedID, reg.Modelo, reg.NodoWorker)
	return nil
}

// GuardarLog inserta un evento de log del cluster
func (c *ClienteMongo) GuardarLog(nivel, nodo, mensaje string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	reg := RegistroLog{
		Timestamp: time.Now().UTC(),
		Nivel:     nivel,
		Nodo:      nodo,
		Mensaje:   mensaje,
	}
	col := c.db.Collection(ColeccionLogs)
	if _, err := col.InsertOne(ctx, reg); err != nil {
		log.Printf("[MongoDB] error guardando log: %v\n", err)
	}
}

// ObtenerPredicciones retorna las últimas N predicciones de un modelo
func (c *ClienteMongo) ObtenerPredicciones(modelo string, limite int) ([]RegistroPrediccion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := c.db.Collection(ColeccionPred)
	filtro := bson.M{}
	if modelo != "" {
		filtro["modelo"] = modelo
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limite))

	cursor, err := col.Find(ctx, filtro, opts)
	if err != nil {
		return nil, fmt.Errorf("[MongoDB] error consultando: %w", err)
	}
	defer cursor.Close(ctx)

	var resultados []RegistroPrediccion
	if err := cursor.All(ctx, &resultados); err != nil {
		return nil, fmt.Errorf("[MongoDB] error decodificando: %w", err)
	}
	return resultados, nil
}

// ContarPredicciones retorna el total de predicciones guardadas
func (c *ClienteMongo) ContarPredicciones() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	col := c.db.Collection(ColeccionPred)
	return col.CountDocuments(ctx, bson.M{})
}
