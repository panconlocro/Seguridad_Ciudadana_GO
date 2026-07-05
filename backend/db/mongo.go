package db

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
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
	ID         string                 `bson:"_id,omitempty" json:"id,omitempty"`
	Timestamp  time.Time              `bson:"timestamp" json:"timestamp"`
	Modelo     string                 `bson:"modelo" json:"modelo"`      // "model1", "model2", "model3"
	NodoWorker string                 `bson:"nodo_worker" json:"nodo_worker"` // ID del nodo que respondió
	Features   map[string]interface{} `bson:"features" json:"features"`    // inputs del usuario
	Resultado  map[string]interface{} `bson:"resultado" json:"resultado"`   // predicción devuelta
	DuracionMs int64                  `bson:"duracion_ms" json:"duracion_ms"` // tiempo de inferencia
}

// RegistroLog representa un evento del cluster
type RegistroLog struct {
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	Nivel     string    `bson:"nivel" json:"nivel"` // "INFO", "ERROR", "WARN"
	Nodo      string    `bson:"nodo" json:"nodo"`
	Mensaje   string    `bson:"mensaje" json:"mensaje"`
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

// ═══════════════════════════════════════════════════════
// GRIDFS — Almacenamiento de Modelos y CSVs
// ═══════════════════════════════════════════════════════

// GuardarArchivoGridFS guarda o sobreescribe un archivo en GridFS
func (c *ClienteMongo) GuardarArchivoGridFS(nombre string, data []byte) error {
	opts := options.GridFSBucket().SetName("archivos")
	bucket, err := gridfs.NewBucket(c.db, opts)
	if err != nil {
		return fmt.Errorf("[GridFS] error creando bucket: %w", err)
	}

	// Borrar versiones anteriores si existen
	cursor, err := bucket.Find(bson.M{"filename": nombre})
	if err == nil {
		type fileInfo struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		for cursor.Next(context.Background()) {
			var f fileInfo
			if err := cursor.Decode(&f); err == nil {
				_ = bucket.Delete(f.ID)
			}
		}
	}

	uploadStream, err := bucket.OpenUploadStream(nombre)
	if err != nil {
		return fmt.Errorf("[GridFS] error abriendo stream: %w", err)
	}
	defer uploadStream.Close()

	if _, err := io.Copy(uploadStream, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("[GridFS] error escribiendo: %w", err)
	}
	return nil
}

// ObtenerArchivoGridFS descarga un archivo completo desde GridFS
func (c *ClienteMongo) ObtenerArchivoGridFS(nombre string) ([]byte, error) {
	opts := options.GridFSBucket().SetName("archivos")
	bucket, err := gridfs.NewBucket(c.db, opts)
	if err != nil {
		return nil, fmt.Errorf("[GridFS] error creando bucket: %w", err)
	}

	var buf bytes.Buffer
	_, err = bucket.DownloadToStreamByName(nombre, &buf)
	if err != nil {
		return nil, fmt.Errorf("[GridFS] error descargando %s: %w", nombre, err)
	}
	return buf.Bytes(), nil
}

// ExisteArchivoGridFS verifica si un archivo está en GridFS
func (c *ClienteMongo) ExisteArchivoGridFS(nombre string) bool {
	opts := options.GridFSBucket().SetName("archivos")
	bucket, err := gridfs.NewBucket(c.db, opts)
	if err != nil {
		return false
	}
	cursor, err := bucket.Find(bson.M{"filename": nombre})
	if err != nil {
		return false
	}
	defer cursor.Close(context.Background())
	return cursor.Next(context.Background())
}

// ═══════════════════════════════════════════════════════
// JOBS DE ENTRENAMIENTO
// ═══════════════════════════════════════════════════════

const ColeccionTraining = "training_jobs"

type TrainingJob struct {
	ID        string    `bson:"_id,omitempty" json:"id,omitempty"`
	Modelo    string    `bson:"modelo" json:"modelo"`
	Estado    string    `bson:"estado" json:"estado"` // "en_progreso", "completado", "error"
	ErrorMsg  string    `bson:"error_msg,omitempty" json:"error_msg,omitempty"`
	CreadoEn  time.Time `bson:"creado_en" json:"creado_en"`
	FinalizEn time.Time `bson:"finaliz_en,omitempty" json:"finaliz_en,omitempty"`
}

func (c *ClienteMongo) CrearTrabajoEntrenamiento(modelo string) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	job := TrainingJob{
		Modelo:   modelo,
		Estado:   "en_progreso",
		CreadoEn: time.Now().UTC(),
	}
	return c.db.Collection(ColeccionTraining).InsertOne(ctx, job)
}

func (c *ClienteMongo) ActualizarTrabajoEntrenamiento(id interface{}, estado, errorMsg string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"estado":     estado,
			"error_msg":  errorMsg,
			"finaliz_en": time.Now().UTC(),
		},
	}
	_, err := c.db.Collection(ColeccionTraining).UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// ═══════════════════════════════════════════════════════
// AUTENTICACIÓN — Manejo de Usuarios en MongoDB
// ═══════════════════════════════════════════════════════

const ColeccionUsuarios = "usuarios"

// RegistroUsuario representa a un usuario en la BD
type RegistroUsuario struct {
	ID           string    `bson:"_id,omitempty"`
	Usuario      string    `bson:"usuario"`
	PasswordHash string    `bson:"password_hash"`
	Rol          string    `bson:"rol"`
	CreadoEn     time.Time `bson:"creado_en"`
}

// CrearUsuario inserta un nuevo usuario en la base de datos
func (c *ClienteMongo) CrearUsuario(usuario, passwordHash, rol string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verificar si el usuario ya existe
	col := c.db.Collection(ColeccionUsuarios)
	count, err := col.CountDocuments(ctx, bson.M{"usuario": usuario})
	if err != nil {
		return fmt.Errorf("error comprobando existencia: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("el usuario '%s' ya existe", usuario)
	}

	reg := RegistroUsuario{
		Usuario:      usuario,
		PasswordHash: passwordHash,
		Rol:          rol,
		CreadoEn:     time.Now().UTC(),
	}

	_, err = col.InsertOne(ctx, reg)
	if err != nil {
		return fmt.Errorf("error insertando usuario: %w", err)
	}
	
	log.Printf("[MongoDB] ✔ Nuevo usuario creado: %s (rol: %s)\n", usuario, rol)
	return nil
}

// ObtenerUsuarioPorNombre busca un usuario por su nombre exacto
func (c *ClienteMongo) ObtenerUsuarioPorNombre(usuario string) (*RegistroUsuario, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := c.db.Collection(ColeccionUsuarios)
	var reg RegistroUsuario
	err := col.FindOne(ctx, bson.M{"usuario": usuario}).Decode(&reg)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No existe
		}
		return nil, fmt.Errorf("error buscando usuario: %w", err)
	}
	return &reg, nil
}

// InicializarAdmin verifica si hay algún admin; si no lo hay, crea uno por defecto.
func (c *ClienteMongo) InicializarAdmin(hash string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := c.db.Collection(ColeccionUsuarios)
	count, _ := col.CountDocuments(ctx, bson.M{"rol": "admin"})
	if count == 0 {
		log.Println("[MongoDB] No se detectó administrador. Creando admin por defecto...")
		_ = c.CrearUsuario("admin", hash, "admin")
	}
}
