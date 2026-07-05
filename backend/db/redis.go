package db

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// ═══════════════════════════════════════════════════════
// CAPA DE CACHÉ — Redis
// Cachea predicciones para evitar recalcular en el cluster
// ═══════════════════════════════════════════════════════

const (
	TTLCache    = 10 * time.Minute
	PrefixoPred = "pred:"
)

// ClienteRedis encapsula la conexión y operaciones de caché
type ClienteRedis struct {
	client *redis.Client
	hits   int64
	misses int64
}

// NuevoClienteRedis crea y verifica la conexión a Redis
func NuevoClienteRedis(uri string) (*ClienteRedis, error) {
	opts, err := redis.ParseURL(uri)
	if err != nil {
		return nil, fmt.Errorf("[Redis] URL inválida: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("[Redis] ping fallido: %w", err)
	}

	log.Printf("[Redis] ✔ Conectado a %s\n", uri)
	return &ClienteRedis{client: client}, nil
}

// generarClave crea una clave Redis determinística para un modelo + features
func (r *ClienteRedis) generarClave(modelo string, features []float64) string {
	parts := make([]string, len(features))
	for i, f := range features {
		parts[i] = fmt.Sprintf("%.6f", f)
	}
	raw := fmt.Sprintf("%s:%s", modelo, strings.Join(parts, ","))
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%s%s:%x", PrefixoPred, modelo, hash[:8])
}

// ObtenerCache busca una predicción en caché
func (r *ClienteRedis) ObtenerCache(modelo string, features []float64) (map[string]interface{}, bool) {
	clave := r.generarClave(modelo, features)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	val, err := r.client.Get(ctx, clave).Result()
	if err == redis.Nil {
		atomic.AddInt64(&r.misses, 1)
		return nil, false
	}
	if err != nil {
		log.Printf("[Redis] error leyendo caché: %v\n", err)
		atomic.AddInt64(&r.misses, 1)
		return nil, false
	}

	var resultado map[string]interface{}
	if err := json.Unmarshal([]byte(val), &resultado); err != nil {
		atomic.AddInt64(&r.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&r.hits, 1)
	log.Printf("[Redis] ✔ Cache HIT: %s\n", clave)
	return resultado, true
}

// CachearPrediccion guarda una predicción en Redis con TTL
func (r *ClienteRedis) CachearPrediccion(modelo string, features []float64, resultado map[string]interface{}) {
	clave := r.generarClave(modelo, features)

	data, err := json.Marshal(resultado)
	if err != nil {
		log.Printf("[Redis] error serializando para caché: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := r.client.Set(ctx, clave, string(data), TTLCache).Err(); err != nil {
		log.Printf("[Redis] error guardando en caché: %v\n", err)
		return
	}
	log.Printf("[Redis] ✔ Cacheado: %s (TTL: %v)\n", clave, TTLCache)
}

// EstadisticasCache retorna métricas del caché
func (r *ClienteRedis) EstadisticasCache() map[string]interface{} {
	hits := atomic.LoadInt64(&r.hits)
	misses := atomic.LoadInt64(&r.misses)
	total := hits + misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dbSize, _ := r.client.DBSize(ctx).Result()

	return map[string]interface{}{
		"hits":       hits,
		"misses":     misses,
		"hit_rate":   fmt.Sprintf("%.2f%%", hitRate),
		"keys_total": dbSize,
	}
}

// Cerrar cierra la conexión con Redis
func (r *ClienteRedis) Cerrar() {
	if err := r.client.Close(); err != nil {
		log.Printf("[Redis] error al cerrar: %v\n", err)
	} else {
		log.Println("[Redis] Conexión cerrada correctamente")
	}
}
