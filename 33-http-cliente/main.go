// Tema 33: Cliente HTTP (net/http.Client)
// Para hacer requests HTTP se usa http.Client. Puntos clave:
//   - El http.DefaultClient (usado por http.Get) NO tiene timeout: evítalo
//     en código real y crea tu propio cliente con Timeout.
//   - SIEMPRE cierra resp.Body, aunque no lo leas.
//   - Un status 404 o 500 NO es un error de Go: hay que revisar StatusCode.
//
// Para no depender de Internet, este ejemplo levanta un servidor local de
// prueba con httptest.NewServer y le hace requests reales por la red local.
// Ejecutar con: go run ./33-http-cliente
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"
)

type Usuario struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
}

// servidorDePrueba simula una API externa con varias rutas
func servidorDePrueba() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /usuarios/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") != "1" {
			http.Error(w, "usuario no encontrado", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Usuario{ID: 1, Nombre: "Ana"})
	})
	mux.HandleFunc("POST /usuarios", func(w http.ResponseWriter, r *http.Request) {
		var u Usuario
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		u.ID = 2
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(u)
	})
	mux.HandleFunc("GET /eco-headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8") // ver tema 32 (XSS)
		fmt.Fprintf(w, "User-Agent=%q Authorization=%q busqueda=%q",
			r.Header.Get("User-Agent"), r.Header.Get("Authorization"), r.URL.Query().Get("q"))
	})
	mux.HandleFunc("GET /lento", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
			fmt.Fprintln(w, "respuesta tardía")
		case <-r.Context().Done(): // el cliente se rindió
		}
	})
	return httptest.NewServer(mux)
}

// obtenerUsuario muestra el flujo completo de un GET que devuelve JSON
func obtenerUsuario(cliente *http.Client, base string, id int) (Usuario, error) {
	resp, err := cliente.Get(fmt.Sprintf("%s/usuarios/%d", base, id))
	if err != nil {
		return Usuario{}, err // error de red: DNS, conexión rechazada, timeout...
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cuerpo, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return Usuario{}, fmt.Errorf("status %d: %s", resp.StatusCode, bytes.TrimSpace(cuerpo))
	}

	var u Usuario
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return Usuario{}, fmt.Errorf("decodificar respuesta: %w", err)
	}
	return u, nil
}

func main() {
	srv := servidorDePrueba()
	defer srv.Close()
	fmt.Println("Servidor de prueba en", srv.URL)

	// Cliente propio con timeout total (conexión + envío + lectura del cuerpo)
	cliente := &http.Client{Timeout: 3 * time.Second}

	// 1) GET + JSON
	fmt.Println("\n1) GET que devuelve JSON")
	u, err := obtenerUsuario(cliente, srv.URL, 1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   usuario: %+v\n", u)

	_, err = obtenerUsuario(cliente, srv.URL, 42)
	fmt.Println("   id inexistente:", err)

	// 2) POST con cuerpo JSON
	fmt.Println("\n2) POST con cuerpo JSON")
	cuerpo, _ := json.Marshal(Usuario{Nombre: "Bruno"})
	resp, err := cliente.Post(srv.URL+"/usuarios", "application/json", bytes.NewReader(cuerpo))
	if err != nil {
		log.Fatal(err)
	}
	var creado Usuario
	err = json.NewDecoder(resp.Body).Decode(&creado)
	resp.Body.Close()
	if err != nil {
		log.Fatal("decodificar respuesta: ", err)
	}
	fmt.Printf("   status %d (%s), creado: %+v\n", resp.StatusCode, http.StatusText(resp.StatusCode), creado)

	// 3) Request personalizado: headers y query params.
	// http.NewRequest da control total (método, headers, cuerpo).
	fmt.Println("\n3) Request con headers y query string")
	params := url.Values{}
	params.Set("q", "go & café") // url.Values se encarga de escapar caracteres
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/eco-headers?"+params.Encode(), nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "curso-go/1.0")
	req.Header.Set("Authorization", "Bearer token-de-ejemplo")
	resp, err = cliente.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	texto, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println("   URL enviada:", req.URL)
	fmt.Println("   el servidor recibió:", string(texto))

	// 4) Timeout del cliente: /lento tarda 2s y este cliente espera 500ms
	fmt.Println("\n4) Timeout del cliente")
	clienteImpaciente := &http.Client{Timeout: 500 * time.Millisecond}
	inicio := time.Now()
	resp, err = clienteImpaciente.Get(srv.URL + "/lento")
	if err == nil { // no debería ocurrir, pero si hay respuesta su cuerpo se cierra
		resp.Body.Close()
	}
	fmt.Printf("   falló tras %v\n", time.Since(inicio).Round(100*time.Millisecond))
	// Los errores de red implementan la interfaz net.Error, cuyo método
	// Timeout() indica si la causa fue un timeout. errors.AsType (Go 1.26+)
	// busca un error de ese tipo dentro de la cadena de errores envueltos.
	if errRed, ok := errors.AsType[net.Error](err); ok && errRed.Timeout() {
		fmt.Println("   fue un timeout:", err)
	}

	// 5) Cancelación con context: útil para cortar un request según la
	// lógica del programa (p. ej. el usuario cerró la conexión entrante).
	fmt.Println("\n5) Timeout por context")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/lento", nil)
	resp, err = cliente.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	fmt.Println("   error:", err)
	fmt.Println("   ¿es context.DeadlineExceeded?", errors.Is(err, context.DeadlineExceeded))
}
