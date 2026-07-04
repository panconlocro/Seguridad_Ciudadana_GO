package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ═══════════════════════════════════════════════════════
// AUTENTICACIÓN JWT — SecurityGO TB2
// Login + Middleware para rutas protegidas
// ═══════════════════════════════════════════════════════

// Clave secreta para firmar tokens JWT (en producción usar variable de entorno)
var jwtSecretKey = []byte("securitygo-2026-secretkey-pc4-tb2")

// Duración del token: 24 horas
const tokenDuracion = 24 * time.Hour

// UsuarioCredenciales representa las credenciales de login
type UsuarioCredenciales struct {
	Usuario  string `json:"usuario"`
	Password string `json:"password"`
}

// Usuarios estáticos (en producción se consultaría MongoDB)
var usuariosValidos = map[string]struct {
	Password string
	Rol      string
}{
	"admin": {Password: "admin123", Rol: "admin"},
	"user":  {Password: "user123", Rol: "user"},
	"rosa":  {Password: "seguridad2026", Rol: "admin"},
}

// ClaimsJWT extiende los claims estándar con campos personalizados
type ClaimsJWT struct {
	Usuario string `json:"usuario"`
	Rol     string `json:"rol"`
	jwt.RegisteredClaims
}

// Login POST /login — autentica y retorna un token JWT
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respError(w, http.StatusMethodNotAllowed, "usar POST")
		return
	}

	var creds UsuarioCredenciales
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		respError(w, http.StatusBadRequest, fmt.Sprintf("JSON inválido: %v", err))
		return
	}

	// Validar credenciales
	usuario, existe := usuariosValidos[creds.Usuario]
	if !existe || usuario.Password != creds.Password {
		respError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}

	// Generar token JWT
	ahora := time.Now()
	claims := ClaimsJWT{
		Usuario: creds.Usuario,
		Rol:     usuario.Rol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(ahora.Add(tokenDuracion)),
			IssuedAt:  jwt.NewNumericDate(ahora),
			Issuer:    "SecurityGO-TB2",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtSecretKey)
	if err != nil {
		respError(w, http.StatusInternalServerError, "error generando token")
		log.Printf("[Auth] error firmando JWT: %v\n", err)
		return
	}

	log.Printf("[Auth] ✔ Login exitoso: %s (rol: %s)\n", creds.Usuario, usuario.Rol)

	respOK(w, map[string]interface{}{
		"token":   tokenStr,
		"usuario": creds.Usuario,
		"rol":     usuario.Rol,
		"expira":  ahora.Add(tokenDuracion).UTC().Format(time.RFC3339),
	})
}

// JWTMiddleware verifica el token JWT en el header Authorization
// Formato esperado: "Bearer <token>"
func JWTMiddleware(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extraer header Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respError(w, http.StatusUnauthorized, "token requerido — header Authorization faltante")
			return
		}

		// Validar formato "Bearer <token>"
		partes := strings.SplitN(authHeader, " ", 2)
		if len(partes) != 2 || strings.ToLower(partes[0]) != "bearer" {
			respError(w, http.StatusUnauthorized, "formato inválido — usar 'Bearer <token>'")
			return
		}

		tokenStr := partes[1]

		// Parsear y validar el token
		claims := &ClaimsJWT{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
			}
			return jwtSecretKey, nil
		})

		if err != nil || !token.Valid {
			respError(w, http.StatusUnauthorized, "token inválido o expirado")
			return
		}

		// Token válido — continuar
		siguiente.ServeHTTP(w, r)
	})
}
