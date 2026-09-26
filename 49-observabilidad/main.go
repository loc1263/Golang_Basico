// Tema 49: Observabilidad (métricas Prometheus, tracing OpenTelemetry)
// Los tres pilares para entender un servicio en producción:
//   - Logs: eventos puntuales (tema 35, slog).
//   - Métricas: números agregados en el tiempo (requests/s, latencia p99,
//     errores). Prometheus las "raspa" (scrape) periódicamente de /metrics.
//   - Trazas: el recorrido de UN request a través de funciones y servicios,
//     dividido en "spans" con duración. OpenTelemetry (OTel) es el estándar.
//
// Unir los tres (p. ej. poner el trace_id en cada log) permite ir de
// "la latencia subió" a "este request, en este servicio, en esta consulta".
//
// Este ejemplo levanta dos servicios HTTP de prueba (frontend -> inventario),
// les hace requests y muestra las métricas y las trazas resultantes.
// Ejecutar con: go run ./49-observabilidad
//
// En un entorno real: las métricas las lee un servidor Prometheus y se ven en
// Grafana; las trazas se envían por OTLP (otlptracegrpc/otlptracehttp) a
// Jaeger, Tempo, Honeycomb, Datadog...
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// ======================= MÉTRICAS =======================

// Tipos de métricas en Prometheus:
//
//	Counter:   solo sube (requests totales, errores)
//	Gauge:     sube y baja (requests en curso, tamaño de cola, memoria)
//	Histogram: distribución de valores en "buckets" (latencias) -> percentiles
var (
	// Registro propio en vez del global: más fácil de testear y controlar
	registro = prometheus.NewRegistry()

	requestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Cantidad de requests HTTP atendidos.",
	}, []string{"servicio", "ruta", "status"}) // labels: dimensiones para filtrar

	requestsEnCurso = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_en_curso",
		Help: "Requests que se están procesando ahora mismo.",
	})

	duracion = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duracion_segundos",
		Help:    "Latencia de los requests.",
		Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25},
	}, []string{"servicio", "ruta"})
)

func init() {
	registro.MustRegister(requestsTotal, requestsEnCurso, duracion)
	// Métricas del runtime de Go (goroutines, GC, memoria) gratis:
	registro.MustRegister(collectors.NewGoCollector())
}

type grabadorStatus struct {
	http.ResponseWriter
	status int
}

func (g *grabadorStatus) WriteHeader(c int) { g.status = c; g.ResponseWriter.WriteHeader(c) }

// ============== MIDDLEWARE: métricas + trazas (tema 34) ==============

func instrumentar(servicio, ruta string, h http.HandlerFunc) http.Handler {
	tracer := otel.Tracer(servicio)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extraer el contexto de traza que viene en el header "traceparent":
		// así el span de este servicio queda como HIJO del span del que llamó.
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := tracer.Start(ctx, r.Method+" "+ruta, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		requestsEnCurso.Inc()
		defer requestsEnCurso.Dec()
		inicio := time.Now()

		g := &grabadorStatus{ResponseWriter: w, status: 200}
		h(g, r.WithContext(ctx))

		duracion.WithLabelValues(servicio, ruta).Observe(time.Since(inicio).Seconds())
		requestsTotal.WithLabelValues(servicio, ruta, fmt.Sprint(g.status)).Inc()
		span.SetAttributes(attribute.Int("http.status_code", g.status))
		if g.status >= 500 {
			span.SetStatus(codes.Error, http.StatusText(g.status))
		}
	})
}

// logConTraza agrega trace_id y span_id a cada log: correlaciona logs y trazas
func logConTraza(ctx context.Context) *slog.Logger {
	sc := trace.SpanContextFromContext(ctx)
	return slog.Default().With("trace_id", sc.TraceID().String()[:8]+"…", "span_id", sc.SpanID().String()[:8]+"…")
}

// ======================= SERVICIOS =======================

// Servicio "inventario": consulta una "base de datos" lenta y a veces falla
func manejarInventario(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Span hijo manual para una operación interna
	_, span := otel.Tracer("inventario").Start(ctx, "consulta SQL")
	span.SetAttributes(attribute.String("db.statement", "SELECT stock FROM productos WHERE id = ?"))
	time.Sleep(time.Duration(5+rand.IntN(40)) * time.Millisecond)
	falla := r.URL.Query().Get("id") == "13"
	if falla {
		err := errors.New("timeout de la base de datos")
		span.RecordError(err) // registrar el error como evento del span
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()

	if falla {
		logConTraza(ctx).Error("fallo consultando stock", "producto", 13)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, "stock=42")
}

// Servicio "frontend": recibe al usuario y llama a inventario por HTTP
func manejarFrontend(urlInventario string) http.HandlerFunc {
	cliente := &http.Client{Timeout: 2 * time.Second}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := r.URL.Query().Get("id")

		ctx, span := otel.Tracer("frontend").Start(ctx, "llamar inventario", trace.WithSpanKind(trace.SpanKindClient))
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, urlInventario+"/stock?id="+id, nil)
		// Inyectar el contexto de traza en los headers salientes (traceparent)
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
		resp, err := cliente.Do(req)
		if err != nil {
			span.End()
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		cuerpo, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		span.End()

		if resp.StatusCode != http.StatusOK {
			logConTraza(ctx).Warn("inventario respondió con error", "status", resp.StatusCode)
			http.Error(w, "inventario no disponible", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8") // evita XSS (tema 32)
		fmt.Fprintf(w, "producto %s: %s", id, cuerpo)
	}
}

// ======================= EXPORTADOR DE RESUMEN =======================

// resumenExporter implementa sdktrace.SpanExporter imprimiendo una línea por
// span. Los exportadores reales (OTLP, Jaeger...) implementan esta misma
// interfaz y envían los spans por red.
type resumenExporter struct{ activo *atomic.Bool }

func (e resumenExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if !e.activo.Load() {
		return nil
	}
	for _, s := range spans {
		padre := "(raíz)"
		if s.Parent().IsValid() {
			padre = "padre=" + s.Parent().SpanID().String()[:6]
		}
		fmt.Printf("   traza=%s span=%s %-13s %-10s %-22s %6v %s\n",
			s.SpanContext().TraceID().String()[:6], s.SpanContext().SpanID().String()[:6], padre,
			s.InstrumentationScope().Name, s.Name(), // Name = nombre del tracer
			s.EndTime().Sub(s.StartTime()).Round(time.Millisecond), s.Status().Code)
	}
	return nil
}
func (resumenExporter) Shutdown(context.Context) error { return nil }

func nuevoTracerProvider(servicio string, exporters ...sdktrace.SpanExporter) *sdktrace.TracerProvider {
	opciones := []sdktrace.TracerProviderOption{
		// El "resource" describe QUIÉN emite las trazas
		sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", servicio))),
		// En producción se suele muestrear (p. ej. 10%) para reducir costos:
		// sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1)))
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}
	for _, e := range exporters {
		// Syncer exporta al terminar cada span (bueno para demos);
		// en producción se usa WithBatcher para enviar en lotes
		opciones = append(opciones, sdktrace.WithSyncer(e))
	}
	return sdktrace.NewTracerProvider(opciones...)
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(g []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})))

	// Además del resumen en consola, guardamos las trazas completas en JSON
	// con el exportador oficial stdouttrace (útil para ver toda la estructura)
	archivo, err := os.Create("49-observabilidad/trazas.json")
	if err != nil {
		log.Fatal(err)
	}
	defer archivo.Close()
	jsonExp, _ := stdouttrace.New(stdouttrace.WithWriter(archivo), stdouttrace.WithPrettyPrint())

	var mostrarSpans atomic.Bool
	mostrarSpans.Store(true)
	tp := nuevoTracerProvider("demo", resumenExporter{activo: &mostrarSpans}, jsonExp)
	// Shutdown vacía los spans pendientes antes de salir
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Println("error cerrando el tracer:", err)
		}
	}()
	otel.SetTracerProvider(tp)
	// W3C Trace Context: el formato estándar del header traceparent
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Levantar los dos servicios
	muxInv := http.NewServeMux()
	muxInv.Handle("GET /stock", instrumentar("inventario", "/stock", manejarInventario))
	inventario := httptest.NewServer(muxInv)
	defer inventario.Close()

	muxFront := http.NewServeMux()
	muxFront.Handle("GET /producto", instrumentar("frontend", "/producto", manejarFrontend(inventario.URL)))
	muxFront.Handle("GET /metrics", promhttp.HandlerFor(registro, promhttp.HandlerOpts{}))
	frontend := httptest.NewServer(muxFront)
	defer frontend.Close()

	fmt.Println("1) Trazas de 2 requests (frontend -> inventario)")
	fmt.Println("   Cada request produce 4 spans con el MISMO id de traza, anidados:")
	for _, id := range []string{"7", "13"} {
		fmt.Printf("\n   GET /producto?id=%s\n", id)
		resp, err := http.Get(frontend.URL + "/producto?id=" + id)
		if err != nil {
			log.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Algunos requests más para que las métricas tengan datos (sin imprimir spans)
	mostrarSpans.Store(false)
	for i := 100; i < 120; i++ {
		resp, _ := http.Get(fmt.Sprintf("%s/producto?id=%d", frontend.URL, i))
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	fmt.Println("\n2) Métricas expuestas en /metrics (formato de texto de Prometheus, filtradas):")
	resp, err := http.Get(frontend.URL + "/metrics")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		l := sc.Text()
		if strings.HasPrefix(l, "http_requests_total") ||
			strings.HasPrefix(l, "http_request_duracion_segundos_bucket{ruta=\"/producto\"") ||
			strings.HasPrefix(l, "http_request_duracion_segundos_count") ||
			strings.HasPrefix(l, "go_goroutines") {
			fmt.Println("  ", l)
		}
	}
	fmt.Println("\n   Consultas PromQL típicas sobre estas métricas:")
	fmt.Println(`   rate(http_requests_total{status=~"5.."}[5m])                  errores por segundo`)
	fmt.Println(`   histogram_quantile(0.99, rate(http_request_duracion_segundos_bucket[5m]))   latencia p99`)
	fmt.Println("\nTrazas completas en formato JSON: 49-observabilidad/trazas.json")
}
