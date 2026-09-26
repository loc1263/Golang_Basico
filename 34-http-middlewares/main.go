// Tema 34: Middlewares HTTP
// Un middleware es una función que recibe un http.Handler y devuelve otro
// http.Handler que "envuelve" al original para hacer algo antes y/o después:
// logging, autenticación, recuperación de panics, CORS, etc.
//
//	func middleware(siguiente http.Handler) http.Handler
//
// Como entrada y salida tienen el mismo tipo, se pueden encadenar.
// Este ejemplo prueba los handlers con httptest.NewRecorder, que simula un
// ResponseWriter sin abrir ningún puerto.
// Ejecutar con: go run ./34-http-middlewares
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"time"
)

type Middleware func(http.Handler) http.Handler

// --- Middleware 1: logging ---

// grabadorStatus envuelve el ResponseWriter para "espiar" el status code,
// ya que http.ResponseWriter no permite leerlo después de escribirlo.
type grabadorStatus struct {
	http.ResponseWriter // embedding: hereda Header(), Write(), etc.
	status              int
}

func (g *grabadorStatus) WriteHeader(code int) {
	g.status = code
	g.ResponseWriter.WriteHeader(code)
}

func Logging(logger *log.Logger) Middleware {
	return func(siguiente http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inicio := time.Now()
			g := &grabadorStatus{ResponseWriter: w, status: http.StatusOK}
			siguiente.ServeHTTP(g, r) // llamar al siguiente eslabón de la cadena
			// Esto se ejecuta DESPUÉS de que el handler terminó
			logger.Printf("%s %s -> %d (%v)", r.Method, r.URL.Path, g.status, time.Since(inicio).Round(time.Microsecond))
		})
	}
}

// --- Middleware 2: recuperar panics ---

// Recuperar evita que un panic en un handler tire abajo la respuesta:
// lo convierte en un 500. (net/http ya evita que caiga el servidor entero,
// pero el cliente recibiría la conexión cortada.)
func Recuperar(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				http.Error(w, "error interno", http.StatusInternalServerError)
			}
		}()
		siguiente.ServeHTTP(w, r)
	})
}

// --- Middleware 3: autenticación simple + context ---

// Tipo propio para la clave del context: evita choques con claves de otros
// paquetes (nunca uses un string suelto como clave).
type claveCtx string

const claveUsuario claveCtx = "usuario"

// tokensValidos simula una base de datos de tokens
var tokensValidos = map[string]string{
	"secreto-ana":   "ana",
	"secreto-bruno": "bruno",
}

// RequiereAuth verifica el header Authorization. Si falla, corta la cadena
// (NO llama a siguiente). Si pasa, guarda el usuario en el context del request.
func RequiereAuth(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		usuario, valido := tokensValidos[token]
		if !ok || !valido {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "no autorizado", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), claveUsuario, usuario)
		siguiente.ServeHTTP(w, r.WithContext(ctx))
	})
}

// --- Middleware 4: header común ---

func HeaderServidor(nombre string) Middleware {
	return func(siguiente http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Servidor", nombre) // antes de llamar al siguiente
			siguiente.ServeHTTP(w, r)
		})
	}
}

// Encadenar aplica los middlewares de modo que el PRIMERO de la lista sea el
// más externo: Encadenar(h, A, B, C) == A(B(C(h))).
func Encadenar(h http.Handler, mws ...Middleware) http.Handler {
	// Se recorren al revés: el último envuelve primero al handler, y el
	// primero queda afuera de todo. slices.Backward (Go 1.23+) itera del
	// final al principio.
	for _, mw := range slices.Backward(mws) {
		h = mw(h)
	}
	return h
}

// --- Handlers finales ---

func publico(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "contenido público")
}

func perfil(w http.ResponseWriter, r *http.Request) {
	// Leer el valor que dejó el middleware. La aserción ", ok" evita un
	// panic si por error esta ruta quedó sin RequiereAuth.
	usuario, _ := r.Context().Value(claveUsuario).(string)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8") // ver tema 32 (XSS)
	fmt.Fprintf(w, "perfil de %s\n", usuario)
}

// roto simula un bug que provoca un panic en medio de un request
// (en código real sería, por ejemplo, acceder a un puntero nil)
func roto(w http.ResponseWriter, r *http.Request) {
	panic("bug inesperado en el handler")
}

func main() {
	logger := log.New(os.Stdout, "   [log] ", 0)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /publico", publico)
	// Un middleware puede aplicarse a una sola ruta...
	mux.Handle("GET /perfil", RequiereAuth(http.HandlerFunc(perfil)))
	mux.HandleFunc("GET /roto", roto)

	// ...o a todo el mux (todas las rutas)
	app := Encadenar(mux, Logging(logger), Recuperar, HeaderServidor("curso-go"))

	pedidos := []struct {
		descripcion string
		ruta        string
		token       string
	}{
		{"Ruta pública", "/publico", ""},
		{"Perfil sin token", "/perfil", ""},
		{"Perfil con token inválido", "/perfil", "falso"},
		{"Perfil con token válido", "/perfil", "secreto-ana"},
		{"Handler que hace panic", "/roto", ""},
		{"Ruta inexistente", "/nada", ""},
	}

	for _, p := range pedidos {
		fmt.Printf("\n%s: GET %s\n", p.descripcion, p.ruta)
		req := httptest.NewRequest(http.MethodGet, p.ruta, nil)
		if p.token != "" {
			req.Header.Set("Authorization", "Bearer "+p.token)
		}
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		fmt.Printf("   status=%d X-Servidor=%q cuerpo=%q\n",
			rec.Code, rec.Header().Get("X-Servidor"), strings.TrimSpace(rec.Body.String()))
	}

	// Orden de ejecución: cada middleware "envuelve" al siguiente como capas
	// de una cebolla. Lo mostramos con middlewares que solo imprimen.
	fmt.Println("\nOrden de ejecución de la cadena:")
	traza := func(nombre string) Middleware {
		return func(sig http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Println("   entra", nombre)
				sig.ServeHTTP(w, r)
				fmt.Println("   sale ", nombre)
			})
		}
	}
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("   -> handler final")
	})
	Encadenar(final, traza("A"), traza("B"), traza("C")).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}
