package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ═══════════════════════════════════════════════════════
// AUTENTICACIÓN JWT — SecurityGO TB2
// Login + Middleware para rutas protegidas
// ═══════════════════════════════════════════════════════

// Clave secreta para firmar tokens JWT (en producción usar variable de entorno)
var jwtSecretKey = []byte("securitygo-2026-secretkey-pc4-tb2")

// Duración del token: 24 horas
const tokenDuracion = 24 * time.Hour

// UsuarioCredenciales representa las credenciales de login/registro
type UsuarioCredenciales struct {
	Usuario  string `json:"usuario"`
	Password string `json:"password"`
}

// ClaimsJWT extiende los claims estándar con campos personalizados
type ClaimsJWT struct {
	Usuario string `json:"usuario"`
	Rol     string `json:"rol"`
	jwt.RegisteredClaims
}

// Register POST /register — crea un nuevo usuario
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respError(w, http.StatusMethodNotAllowed, "usar POST")
		return
	}

	var creds UsuarioCredenciales
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		respError(w, http.StatusBadRequest, fmt.Sprintf("JSON inválido: %v", err))
		return
	}

	if len(creds.Usuario) < 3 || len(creds.Password) < 6 {
		respError(w, http.StatusBadRequest, "usuario min 3 chars, password min 6 chars")
		return
	}

	// Encriptar contraseña
	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		respError(w, http.StatusInternalServerError, "error procesando contraseña")
		return
	}

	// Guardar en BD con rol "user" por defecto
	err = h.mongo.CrearUsuario(creds.Usuario, string(hash), "user")
	if err != nil {
		if strings.Contains(err.Error(), "ya existe") {
			respError(w, http.StatusConflict, "el usuario ya existe")
		} else {
			respError(w, http.StatusInternalServerError, "error en base de datos")
			log.Printf("[Auth] Error en registro: %v\n", err)
		}
		return
	}

	respOK(w, map[string]string{"mensaje": "usuario creado exitosamente"})
}

// Login POST /login — autentica consultando MongoDB y retorna un token JWT
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

	// Buscar en BD
	usuario, err := h.mongo.ObtenerUsuarioPorNombre(creds.Usuario)
	if err != nil {
		respError(w, http.StatusInternalServerError, "error de base de datos")
		return
	}
	if usuario == nil {
		respError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}

	// Comparar hash
	if err := bcrypt.CompareHashAndPassword([]byte(usuario.PasswordHash), []byte(creds.Password)); err != nil {
		respError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}

	// Generar token JWT
	ahora := time.Now()
	claims := ClaimsJWT{
		Usuario: usuario.Usuario,
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

	log.Printf("[Auth] ✔ Login exitoso: %s (rol: %s)\n", usuario.Usuario, usuario.Rol)

	respOK(w, map[string]interface{}{
		"token":   tokenStr,
		"usuario": usuario.Usuario,
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
