// Tema 32: Servidor HTTP básico (net/http)
// La librería estándar trae un servidor HTTP listo para producción. Las piezas:
//   - http.Handler: cualquier tipo con el método ServeHTTP(w, r)
//   - http.HandlerFunc: adapta una función común a http.Handler
//   - http.ServeMux: el "router" que elige qué handler atiende cada ruta
//
// Desde Go 1.22 los patrones del ServeMux aceptan método y comodines:
// "GET /tareas/{id}".
//
// Ejecutar con: go run ./32-http-servidor
// y probar desde otra terminal, por ejemplo:
//
//	curl http://localhost:8080/
//	curl http://localhost:8080/tareas
//	curl http://localhost:8080/tareas/1
//	curl -X POST -d "{\"titulo\":\"Aprender Go\"}" http://localhost:8080/tareas
//	curl -X DELETE http://localhost:8080/tareas/1
//	curl "http://localhost:8080/saludo?nombre=Ana"
//
// Detener con Ctrl+C.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Tarea struct {
	ID         int    `json:"id"`
	Titulo     string `json:"titulo"`
	Completada bool   `json:"completada"`
}

// almacen guarda las tareas en memoria. El servidor atiende cada request en
// su propia goroutine, así que el acceso concurrente se protege con un mutex.
type almacen struct {
	mu        sync.Mutex
	tareas    map[int]Tarea
	siguiente int
}

func nuevoAlmacen() *almacen {
	return &almacen{tareas: make(map[int]Tarea), siguiente: 1}
}

// responderJSON centraliza cómo se escriben las respuestas JSON.
// Orden importante: primero headers, luego WriteHeader, luego el cuerpo.
func responderJSON(w http.ResponseWriter, status int, datos any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(datos)
}

func responderError(w http.ResponseWriter, status int, mensaje string) {
	responderJSON(w, status, map[string]string{"error": mensaje})
}

// Los handlers son métodos de *almacen para tener acceso a los datos
// (alternativa limpia a usar variables globales).

func (a *almacen) listar(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	lista := make([]Tarea, 0, len(a.tareas))
	for id := 1; id < a.siguiente; id++ { // recorrer por id para un orden estable
		if t, ok := a.tareas[id]; ok {
			lista = append(lista, t)
		}
	}
	responderJSON(w, http.StatusOK, lista)
}

func (a *almacen) obtener(w http.ResponseWriter, r *http.Request) {
	// r.PathValue lee el comodín {id} del patrón de la ruta
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		responderError(w, http.StatusBadRequest, "id inválido")
		return
	}
	a.mu.Lock()
	t, ok := a.tareas[id]
	a.mu.Unlock()
	if !ok {
		responderError(w, http.StatusNotFound, "tarea no encontrada")
		return
	}
	responderJSON(w, http.StatusOK, t)
}

func (a *almacen) crear(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		Titulo string `json:"titulo"`
	}
	// Limitar el tamaño del cuerpo evita que un cliente envíe datos enormes
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		responderError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return
	}
	if entrada.Titulo == "" {
		responderError(w, http.StatusUnprocessableEntity, "el título es obligatorio")
		return
	}

	a.mu.Lock()
	t := Tarea{ID: a.siguiente, Titulo: entrada.Titulo}
	a.tareas[t.ID] = t
	a.siguiente++
	a.mu.Unlock()

	w.Header().Set("Location", fmt.Sprintf("/tareas/%d", t.ID))
	responderJSON(w, http.StatusCreated, t)
}

func (a *almacen) eliminar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		responderError(w, http.StatusBadRequest, "id inválido")
		return
	}
	a.mu.Lock()
	_, ok := a.tareas[id]
	delete(a.tareas, id)
	a.mu.Unlock()
	if !ok {
		responderError(w, http.StatusNotFound, "tarea no encontrada")
		return
	}
	w.WriteHeader(http.StatusNoContent) // 204: éxito sin cuerpo
}

// saludo muestra cómo leer parámetros de la query string (?nombre=Ana)
func saludo(w http.ResponseWriter, r *http.Request) {
	nombre := r.URL.Query().Get("nombre")
	if nombre == "" {
		nombre = "desconocido"
	}
	// Seguridad: "nombre" lo controla el usuario. Si no se indica el
	// Content-Type, Go lo adivina a partir del contenido y podría tratarlo
	// como HTML, lo que permitiría inyectar scripts (XSS). Declararlo como
	// texto plano lo evita. (Para responder HTML se usa html/template, que
	// escapa los datos automáticamente.)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// fmt.Fprintf escribe directamente en el ResponseWriter (es un io.Writer)
	fmt.Fprintf(w, "¡Hola, %s! Método: %s, ruta: %s\n", nombre, r.Method, r.URL.Path)
}

// contador implementa http.Handler con un struct propio en vez de una función
type contador struct {
	mu      sync.Mutex
	visitas int
}

func (c *contador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	c.visitas++
	n := c.visitas
	c.mu.Unlock()
	fmt.Fprintf(w, "Esta ruta fue visitada %d veces\n", n)
}

func main() {
	datos := nuevoAlmacen()
	mux := http.NewServeMux()

	// "GET /{$}" coincide SOLO con "/" exacto. Sin {$}, "/" atraparía
	// cualquier ruta no registrada.
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "API de tareas. Rutas: GET/POST /tareas, GET/DELETE /tareas/{id}, GET /saludo, GET /visitas")
	})
	mux.HandleFunc("GET /tareas", datos.listar)
	mux.HandleFunc("POST /tareas", datos.crear)
	mux.HandleFunc("GET /tareas/{id}", datos.obtener)
	mux.HandleFunc("DELETE /tareas/{id}", datos.eliminar)
	mux.HandleFunc("GET /saludo", saludo)
	mux.Handle("GET /visitas", &contador{}) // Handle recibe un http.Handler

	// Si el método no coincide (p. ej. PUT /tareas) el mux responde 405
	// automáticamente; si la ruta no existe, 404.

	// Crear el http.Server explícitamente permite configurar timeouts.
	// http.ListenAndServe(":8080", mux) funciona, pero NO tiene timeouts.
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Println("Servidor escuchando en http://localhost:8080 (Ctrl+C para salir)")
	// ListenAndServe bloquea hasta que el servidor se detiene
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
