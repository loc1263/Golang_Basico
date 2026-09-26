// Tema 44: Autenticación y seguridad (bcrypt, JWT, TLS)
// Tres piezas básicas para asegurar una aplicación:
//  1. bcrypt (golang.org/x/crypto/bcrypt): guardar contraseñas como HASH
//     lento y con "sal". Nunca en texto plano ni con MD5/SHA sin más.
//  2. JWT (github.com/golang-jwt/jwt/v5): tokens firmados que el cliente
//     envía en cada request (Authorization: Bearer ...). El servidor verifica
//     la firma sin guardar sesiones.
//  3. TLS (crypto/tls): cifrar el tráfico HTTP -> HTTPS.
//
// Ejecutar con: go run ./44-seguridad
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ===================== 1) BCRYPT =====================

func ejemploBcrypt() {
	fmt.Println("===== 1) Hash de contraseñas con bcrypt =====")
	password := []byte("mi-clave-segura-123")

	// El costo define cuántas rondas: +1 duplica el tiempo. DefaultCost=10.
	// La lentitud es intencional: frena ataques de fuerza bruta.
	inicio := time.Now()
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("hash (%v): %s\n", time.Since(inicio).Round(time.Millisecond), hash)
	// Formato: $2a$<costo>$<sal de 22 chars><hash>. La sal va incluida.

	// Mismo password -> hash distinto cada vez (sal aleatoria)
	hash2, _ := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	fmt.Println("¿dos hashes iguales?", string(hash) == string(hash2))

	// Verificar: se compara el password recibido contra el hash guardado
	fmt.Println("clave correcta:  ", bcrypt.CompareHashAndPassword(hash, password) == nil)
	err = bcrypt.CompareHashAndPassword(hash, []byte("otra-clave"))
	fmt.Println("clave incorrecta:", errors.Is(err, bcrypt.ErrMismatchedHashAndPassword))

	costo, _ := bcrypt.Cost(hash)
	fmt.Println("costo guardado en el hash:", costo)
	// Límite: bcrypt solo usa los primeros 72 bytes del password
}

// ===================== 2) JWT =====================

// En producción la clave sale de una variable de entorno / gestor de
// secretos (tema 36), nunca del código.
var claveJWT = []byte("clave-secreta-de-al-menos-32-bytes!!")

// Claims propios + los estándar (exp, iat, sub, iss...) de RegisteredClaims
type Claims struct {
	Rol string `json:"rol"`
	jwt.RegisteredClaims
}

func generarToken(usuario, rol string, duracion time.Duration) (string, error) {
	ahora := time.Now()
	claims := Claims{
		Rol: rol,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   usuario,
			Issuer:    "curso-go",
			IssuedAt:  jwt.NewNumericDate(ahora),
			ExpiresAt: jwt.NewNumericDate(ahora.Add(duracion)),
		},
	}
	// HS256 = HMAC-SHA256 con clave simétrica. Para que otros servicios
	// verifiquen sin poder firmar, se usa RS256/ES256 (clave pública/privada).
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(claveJWT)
}

func validarToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (any, error) { return claveJWT, nil },
		// MUY importante: fijar los algoritmos aceptados. Si no, un atacante
		// podría enviar un token con alg "none" u otro algoritmo.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer("curso-go"),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func ejemploJWT() {
	fmt.Println("\n===== 2) JWT =====")
	token, err := generarToken("ana", "admin", 15*time.Minute)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("token:", token)

	// Un JWT son 3 partes base64url separadas por puntos:
	// header.payload.firma. El payload NO está cifrado: cualquiera lo lee.
	// Nunca pongas datos secretos en un JWT.
	partes := strings.Split(token, ".")
	payload, _ := base64.RawURLEncoding.DecodeString(partes[1])
	fmt.Println("payload legible por cualquiera:", string(payload))

	claims, err := validarToken(token)
	fmt.Printf("válido: usuario=%s rol=%s expira=%s err=%v\n",
		claims.Subject, claims.Rol, claims.ExpiresAt.Format(time.TimeOnly), err)

	// Token manipulado: cambiamos el payload para hacernos "admin" de otro usuario
	falso := partes[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"rol":"admin","sub":"mallory","iss":"curso-go","exp":9999999999}`)) + "." + partes[2]
	_, err = validarToken(falso)
	fmt.Println("manipulado:", errors.Is(err, jwt.ErrTokenSignatureInvalid), "->", err)

	// Token vencido
	vencido, _ := generarToken("ana", "admin", -time.Minute)
	_, err = validarToken(vencido)
	fmt.Println("vencido:   ", errors.Is(err, jwt.ErrTokenExpired), "->", err)
}

// ===================== 3) TLS / HTTPS =====================

// certificadoAutofirmado crea en memoria un certificado para 127.0.0.1.
// En producción el certificado lo emite una CA (p. ej. Let's Encrypt, que en
// Go se automatiza con golang.org/x/crypto/acme/autocert).
func certificadoAutofirmado() (tls.Certificate, *x509.Certificate, error) {
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	plantilla := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Curso Go"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &clave.PublicKey, clave)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	cert, _ := x509.ParseCertificate(der)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: clave}, cert, nil
}

func ejemploTLS() {
	fmt.Println("\n===== 3) HTTPS con TLS + ruta protegida con JWT =====")
	tlsCert, x509Cert, err := certificadoAutofirmado()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		// En una app real: buscar el usuario y comparar con bcrypt
		usuario, clave, ok := r.BasicAuth()
		if !ok || usuario != "ana" || subtle.ConstantTimeCompare([]byte(clave), []byte("secreto")) != 1 {
			http.Error(w, "credenciales inválidas", http.StatusUnauthorized)
			return
		}
		token, err := generarToken(usuario, "admin", time.Hour)
		if err != nil {
			http.Error(w, "error interno", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, token)
	})
	mux.HandleFunc("GET /privado", func(w http.ResponseWriter, r *http.Request) {
		token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		claims, err := validarToken(token)
		if err != nil {
			http.Error(w, "token inválido", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8") // evita XSS (tema 32)
		fmt.Fprintf(w, "hola %s, tu rol es %s (conexión %s)", claims.Subject, claims.Rol, tls.VersionName(r.TLS.Version))
	})

	srv := httptest.NewUnstartedServer(mux)
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12, // no aceptar versiones viejas e inseguras
	}
	// El servidor registra cada handshake fallido; lo silenciamos porque
	// abajo provocamos uno a propósito
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	defer srv.Close()
	fmt.Println("servidor HTTPS en", srv.URL)

	// Cliente que confía SOLO en nuestro certificado (en vez de desactivar
	// la verificación con InsecureSkipVerify, que nunca debe usarse en producción)
	confiables := x509.NewCertPool()
	confiables.AddCert(x509Cert)
	cliente := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: confiables}},
	}

	// Un cliente sin nuestro certificado rechaza la conexión
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(srv.URL + "/privado")
	if err == nil { // no debería ocurrir: el certificado no es de confianza
		resp.Body.Close()
		log.Fatal("se esperaba un error de certificado")
	}
	fmt.Println("cliente sin el certificado:", err != nil, "->", strings.SplitN(err.Error(), ": ", 3)[2])

	leer := func(resp *http.Response, err error) string {
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return fmt.Sprintf("%d %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/privado", nil)
	fmt.Println("GET /privado sin token:", leer(cliente.Do(req)))

	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/login", nil)
	req.SetBasicAuth("ana", "secreto")
	resp, err = cliente.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	tokenBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println("POST /login -> token de", len(tokenBytes), "caracteres")

	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/privado", nil)
	req.Header.Set("Authorization", "Bearer "+string(tokenBytes))
	fmt.Println("GET /privado con token:", leer(cliente.Do(req)))
}

// ============ Extras: aleatoriedad segura y HMAC ============

func ejemploExtras() {
	fmt.Println("\n===== Extras =====")
	// Para tokens, IDs de sesión o claves: crypto/rand, NUNCA math/rand
	b := make([]byte, 16)
	rand.Read(b)
	fmt.Println("token aleatorio seguro:", hex.EncodeToString(b))
	fmt.Println("rand.Text() (Go 1.24+): ", rand.Text()) // texto aleatorio seguro en base32

	// HMAC: firmar un mensaje para detectar modificaciones (p. ej. webhooks)
	firmar := func(msg string) []byte {
		m := hmac.New(sha256.New, claveJWT)
		m.Write([]byte(msg))
		return m.Sum(nil)
	}
	firma := firmar("monto=100")
	// hmac.Equal compara en tiempo constante (evita ataques de temporización)
	fmt.Println("firma válida:    ", hmac.Equal(firma, firmar("monto=100")))
	fmt.Println("mensaje alterado:", hmac.Equal(firma, firmar("monto=999")))
}

func main() {
	ejemploBcrypt()
	ejemploJWT()
	ejemploTLS()
	ejemploExtras()
}
