// Tema 51: Dockerización de aplicaciones Go
// Go compila a un único binario estático: ideal para contenedores mínimos.
// Una imagen Go bien hecha pesa pocos MB (contra cientos con un runtime de
// Node, Python o Java), arranca en milisegundos y tiene muy poca superficie
// de ataque.
//
// Archivos de este tema:
//
//	Dockerfile               build multi-etapa (compilar en una imagen, ejecutar en otra mínima)
//	Dockerfile.dockerignore  qué NO enviar al build (BuildKit lo asocia a este Dockerfile)
//	compose.yaml             levantar el servicio con docker compose
//	main.go                  servicio preparado para contenedores (este archivo)
//
// Construir y ejecutar (desde la RAÍZ del proyecto, porque go.mod está ahí):
//
//	docker build -f 51-docker/Dockerfile -t curso-go/servicio:1.0 --build-arg VERSION=1.0 .
//	docker run --rm -p 8084:8080 -e APP_MENSAJE="hola desde Docker" curso-go/servicio:1.0
//	curl http://localhost:8084/  y  http://localhost:8084/salud
//	docker compose -f 51-docker/compose.yaml up --build
//
// Sin Docker también funciona: go run ./51-docker
//
// Lo que un servicio necesita para funcionar bien en un contenedor:
//  1. Configuración por variables de entorno (tema 36), puerto incluido.
//  2. Logs a stdout/stderr, en JSON (tema 35): Docker/Kubernetes los recolectan.
//  3. Endpoint de salud para healthchecks / liveness / readiness probes.
//  4. Apagado ordenado (graceful shutdown) al recibir SIGTERM: `docker stop`
//     manda SIGTERM y, si el proceso no termina en 10s, SIGKILL.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
)

// version se inyecta al compilar: go build -ldflags "-X main.version=1.0"
// (el Dockerfile lo hace con el build-arg VERSION). Por defecto "dev".
var version = "dev"

func main() {
	// Modo healthcheck: la imagen final no tiene curl ni shell, así que el
	// propio binario sirve para el HEALTHCHECK del Dockerfile.
	chequear := flag.Bool("healthcheck", false, "consultar /salud y salir con código 0 o 1")
	flag.Parse()

	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8080"
	}
	if *chequear {
		os.Exit(healthcheck(puerto))
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	mensaje := os.Getenv("APP_MENSAJE")
	if mensaje == "" {
		mensaje = "hola"
	}
	hostname, _ := os.Hostname() // en Docker: el ID corto del contenedor

	var listo atomic.Bool // readiness: false mientras arranca y durante el apagado
	inicio := time.Now()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"mensaje":  mensaje,
			"version":  version,
			"host":     hostname,
			"go":       runtime.Version(),
			"os/arch":  runtime.GOOS + "/" + runtime.GOARCH,
			"uptime_s": int(time.Since(inicio).Seconds()),
		})
	})
	// Liveness: ¿el proceso responde? Readiness: ¿puede recibir tráfico?
	mux.HandleFunc("GET /salud", func(w http.ResponseWriter, r *http.Request) {
		if !listo.Load() {
			http.Error(w, "no listo", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /lento", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second) // para probar que el apagado espera a los requests en curso
		fmt.Fprintln(w, "terminé")
	})

	srv := &http.Server{
		Addr:              ":" + puerto, // escuchar en todas las interfaces, no solo localhost
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// signal.NotifyContext cancela ctx al recibir SIGINT (Ctrl+C) o SIGTERM (docker stop)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("servidor iniciado", "puerto", puerto, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("error del servidor", "err", err)
			os.Exit(1)
		}
	}()
	listo.Store(true)

	<-ctx.Done() // esperar la señal
	logger.Info("señal recibida, apagando ordenadamente...")
	listo.Store(false) // dejar de recibir tráfico nuevo del balanceador

	// Shutdown deja de aceptar conexiones y espera a que terminen las activas,
	// con un límite menor al período de gracia de Docker (10s)
	ctxApagado, cancelar := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelar()
	if err := srv.Shutdown(ctxApagado); err != nil {
		logger.Error("apagado forzado", "err", err)
		os.Exit(1)
	}
	logger.Info("servidor detenido")
}

func healthcheck(puerto string) int {
	cliente := &http.Client{Timeout: 2 * time.Second}
	resp, err := cliente.Get("http://127.0.0.1:" + puerto + "/salud")
	if err != nil || resp.StatusCode != http.StatusOK {
		return 1
	}
	resp.Body.Close()
	return 0
}
