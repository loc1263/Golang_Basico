// Tema 50: Arquitectura limpia / hexagonal (puertos y adaptadores)
// Objetivo: que la lógica de negocio no dependa de detalles técnicos
// (base de datos, HTTP, proveedores externos), para poder cambiarlos y
// testear el núcleo de forma aislada.
//
//	           adaptadores de ENTRADA            adaptadores de SALIDA
//	           (httpapi, CLI, gRPC...)           (memoria, sqlite, consola...)
//	                     │                                  ▲
//	                     ▼                                  │ implementan
//	           ┌──────────────────────────────────────────────────┐
//	           │ aplicacion: casos de uso + PUERTOS (interfaces)  │
//	           │        ┌──────────────────────────────┐          │
//	           │        │ dominio: entidades y reglas  │          │
//	           │        └──────────────────────────────┘          │
//	           └──────────────────────────────────────────────────┘
//	Regla de dependencia: los imports apuntan SIEMPRE hacia adentro.
//	dominio no importa nada; aplicacion importa dominio; adaptadores importan
//	aplicacion/dominio; main importa todo y "cablea" las piezas.
//
// Estructura:
//
//	internal/dominio/            Pedido, estados, reglas, errores de dominio
//	internal/aplicacion/         ServicioPedidos (casos de uso) y puertos (+ tests)
//	internal/adaptadores/memoria RepositorioPedidos en un map
//	internal/adaptadores/sqlite  RepositorioPedidos con database/sql
//	internal/adaptadores/consola Notificador que imprime
//	internal/adaptadores/httpapi API REST sobre los casos de uso
//	main.go                      composition root: crea y conecta todo
//
// "internal/" es especial en Go: esos paquetes solo pueden importarse desde
// dentro de 50-arquitectura-limpia/, el compilador lo impide desde afuera.
//
// Ejecutar:     go run ./50-arquitectura-limpia                (repositorio en memoria)
//
//	go run ./50-arquitectura-limpia -repo=sqlite   (mismo programa, otro adaptador)
//	go run ./50-arquitectura-limpia -servir        (API en http://localhost:8083)
//
// Tests:        go test ./50-arquitectura-limpia/...
//
// Advertencia: en proyectos chicos tantas capas pueden ser excesivas. Go
// favorece la simplicidad: empieza simple y separa capas cuando el dominio
// lo justifique.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"golang-basico/50-arquitectura-limpia/internal/adaptadores/consola"
	"golang-basico/50-arquitectura-limpia/internal/adaptadores/httpapi"
	"golang-basico/50-arquitectura-limpia/internal/adaptadores/memoria"
	"golang-basico/50-arquitectura-limpia/internal/adaptadores/sqlite"
	"golang-basico/50-arquitectura-limpia/internal/aplicacion"
)

func main() {
	tipoRepo := flag.String("repo", "memoria", "adaptador de persistencia: memoria | sqlite")
	servir := flag.Bool("servir", false, "levantar la API en :8083 en vez de ejecutar la demo")
	flag.Parse()

	// ===== Composition root: el ÚNICO lugar que conoce las implementaciones =====
	var repo aplicacion.RepositorioPedidos
	switch *tipoRepo {
	case "memoria":
		repo = memoria.NuevoRepositorio()
	case "sqlite":
		r, err := sqlite.NuevoRepositorio(":memory:")
		if err != nil {
			log.Fatal(err)
		}
		defer r.Close()
		repo = r
	default:
		log.Fatalf("repositorio desconocido %q", *tipoRepo)
	}

	var contador atomic.Int64
	nuevoID := func() string { return fmt.Sprintf("P-%03d", contador.Add(1)) }

	svc := aplicacion.NuevoServicioPedidos(repo, consola.Notificador{Salida: os.Stdout}, time.Now, nuevoID)
	api := httpapi.NuevoHandler(svc)

	if *servir {
		fmt.Println("API en http://localhost:8083 (repositorio:", *tipoRepo+")")
		fmt.Println(`  curl -X POST -d "{\"cliente\":\"ana\",\"items\":[{\"producto\":\"libro\",\"cantidad\":2,\"precio\":60}]}" http://localhost:8083/pedidos`)
		// Servidor con timeouts (tema 32): http.ListenAndServe no los tiene
		srv := &http.Server{
			Addr:              ":8083",
			Handler:           api,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
		}
		log.Fatal(srv.ListenAndServe())
	}

	// ===== Demo: requests HTTP reales contra el adaptador de entrada =====
	srv := httptest.NewServer(api)
	defer srv.Close()
	fmt.Printf("Usando repositorio: %s\n", *tipoRepo)

	llamar := func(metodo, ruta, cuerpo string) {
		req, _ := http.NewRequestWithContext(context.Background(), metodo, srv.URL+ruta, bytes.NewBufferString(cuerpo))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		fmt.Printf("%s %s -> %d\n   %s\n", metodo, ruta, resp.StatusCode, strings.TrimSpace(string(b)))
	}

	fmt.Println("\n--- Crear pedidos ---")
	llamar("POST", "/pedidos", `{"cliente":"ana","items":[{"producto":"libro","cantidad":2,"precio":60}]}`)
	llamar("POST", "/pedidos", `{"cliente":"bruno","items":[{"producto":"lápiz","cantidad":3,"precio":1.5}]}`)
	llamar("POST", "/pedidos", `{"cliente":"carla","items":[]}`)

	fmt.Println("\n--- Transiciones de estado (reglas del dominio) ---")
	llamar("POST", "/pedidos/P-001/pagar", "")
	llamar("POST", "/pedidos/P-001/pagar", "") // pagar dos veces: 409
	llamar("POST", "/pedidos/P-002/cancelar", "")
	llamar("GET", "/pedidos/P-999", "")

	fmt.Println("\n--- Listado final ---")
	llamar("GET", "/pedidos", "")
}
