# Golang Básico

Ejemplos didácticos para aprender Go desde cero hasta temas de producción.
Cada tema es un programa independiente, funcional y comentado paso a paso,
ubicado en su propia carpeta. Los comentarios de cada `main.go` explican el
**porqué** de cada línea, así que el código está pensado para leerse de
arriba abajo.

El recorrido tiene tres niveles:

| Nivel | Temas | Contenido |
|-------|-------|-----------|
| Básico | 1-29 | El lenguaje y la librería estándar |
| Intermedio | 30-41 | Bases de datos, HTTP, configuración, dependencias, testing |
| Avanzado | 42-53 | Arquitectura, rendimiento, seguridad, despliegue y microservicios |

## Requisitos

- **Go 1.27 o superior** (`go version` para verificar). El proyecto usa
  novedades recientes del lenguaje y de la librería estándar (ver
  [Novedades de Go usadas](#novedades-de-go-usadas)).
- Opcional: **Docker** para el tema 51 y un compilador de C (gcc) para el
  detector de carreras `-race` en Windows (temas 47 y 52).

## Cómo ejecutar cada tema

Todos los comandos se ejecutan desde la **raíz del proyecto**:

```
go run ./01-variables
go run ./02-condicionales
...
```

Reemplaza el nombre de la carpeta por el tema que quieras ejecutar.

Otros comandos útiles:

```
go test ./...         ejecuta todos los tests del proyecto
go vet ./...          análisis estático: detecta errores comunes
go build ./...        compila todo sin ejecutar
```

## Calidad del código

Todo el proyecto pasa sin avisos las herramientas estándar del ecosistema Go:

| Herramienta | Qué revisa | Comando |
|-------------|------------|---------|
| `gofmt` | formato oficial del código | `gofmt -l .` (no debe listar nada) |
| `go vet` | errores comunes detectados por el compilador | `go vet ./...` |
| [staticcheck](https://staticcheck.dev) | bugs, código muerto, APIs obsoletas | `staticcheck ./...` |
| [golangci-lint](https://golangci-lint.run) | decenas de linters, incluido gosec (seguridad) | `golangci-lint run ./...` |
| [govulncheck](https://go.dev/doc/security/vuln/) | vulnerabilidades conocidas en las dependencias | `govulncheck ./...` |

Instalación de las tres últimas:

```
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

La configuración de golangci-lint está en [`.golangci.yml`](.golangci.yml).
Como el proyecto es didáctico, algunos ejemplos muestran a propósito una
forma "no ideal" para compararla con la correcta; esas reglas están
desactivadas y cada exclusión explica su motivo en el archivo.

La primera vez que ejecutes un tema con dependencias externas (a partir del
tema 30), Go las descarga automáticamente. También puedes descargarlas antes
con `go mod download`.

---

## Índice: Go Básico

| # | Tema | Carpeta |
|---|------|---------|
| 1 | Variables y tipos básicos | [01-variables](01-variables) |
| 2 | Condicionales (if/else, switch) | [02-condicionales](02-condicionales) |
| 3 | Ciclos (for, for-range) | [03-ciclos](03-ciclos) |
| 4 | Arrays | [04-arrays](04-arrays) |
| 5 | Slices y el paquete `slices` | [05-slices](05-slices) |
| 6 | Maps y el paquete `maps` | [06-maps](06-maps) |
| 7 | Structs | [07-structs](07-structs) |
| 8 | Funciones (múltiples retornos, variádicas, closures) | [08-funciones](08-funciones) |
| 9 | Punteros | [09-punteros](09-punteros) |
| 10 | Métodos | [10-metodos](10-metodos) |
| 11 | Interfaces | [11-interfaces](11-interfaces) |
| 12 | Manejo de errores | [12-errores](12-errores) |
| 13 | defer, panic y recover | [13-defer-panic-recover](13-defer-panic-recover) |
| 14 | Goroutines y Channels (concurrencia básica) | [14-goroutines-channels](14-goroutines-channels) |
| 15 | Strings, Runes y Bytes | [15-strings-runes-bytes](15-strings-runes-bytes) |
| 16 | Constantes e iota | [16-constantes-iota](16-constantes-iota) |
| 17 | Generics | [17-generics](17-generics) |
| 18 | Testing (`testing`, `go test`) | [18-testing](18-testing) |
| 19 | JSON (v1 y v2) y archivos | [19-json-archivos](19-json-archivos) |
| 20 | select y sync.Mutex | [20-select-mutex](20-select-mutex) |
| 21 | Paquetes propios y visibilidad | [21-paquetes-propios](21-paquetes-propios) |
| 22 | Ordenar slices (`slices` y `sort`) | [22-sort](22-sort) |
| 23 | context.Context | [23-context](23-context) |
| 24 | Paquete time | [24-time](24-time) |
| 25 | Expresiones regulares (regexp) | [25-regexp](25-regexp) |
| 26 | Argumentos de línea de comandos (flag) | [26-flag](26-flag) |
| 27 | Interfaces estándar (fmt.Stringer, io.Reader, io.Writer) | [27-interfaces-estandar](27-interfaces-estandar) |
| 28 | Labels en loops y goto | [28-labels-goto](28-labels-goto) |
| 29 | Reflection | [29-reflection](29-reflection) |

### Orden sugerido de estudio

El orden de la tabla está pensado como progresión, pero también se puede
estudiar por bloques:

1. **Fundamentos** (1-3): variables, condicionales, ciclos.
2. **Estructuras de datos** (4-7): arrays, slices, maps, structs.
3. **Funciones y punteros** (8-10): funciones, punteros, métodos.
4. **Abstracción y errores** (11-13): interfaces, manejo de errores, defer/panic/recover.
5. **Concurrencia** (14, 20, 23): goroutines/channels, select/mutex, context.
6. **Texto y datos** (15-16, 19, 24-25): strings/runes/bytes, constantes/iota, JSON/archivos, time, regexp.
7. **Herramientas del lenguaje** (17-18, 21-22, 26-29): generics, testing, paquetes propios, ordenamiento, flag, interfaces estándar, labels/goto, reflection.

Notas:
- Tema 18 (testing): lo importante son los tests, así que usa
  `go test -v ./18-testing` además de `go run`.
- Tema 21 (paquetes propios): importa el paquete ubicado en
  `21-paquetes-propios/matematica`.
- Tema 26 (flag): prueba pasando argumentos, por ejemplo
  `go run ./26-flag -nombre=Ana -edad=25 -activo=false extra1 extra2`
  o `go run ./26-flag -h` para ver la ayuda generada.

---

## Índice: Go Intermedio

Temas que aplican lo aprendido a problemas más realistas: persistencia,
comunicación por red y organización de proyectos.

| # | Tema | Carpeta | Notas |
|---|------|---------|-------|
| 30 | `database/sql` básico (CRUD con SQLite embebido) | [30-database-sql](30-database-sql) | SQLite en Go puro (`modernc.org/sqlite`), sin servidor ni gcc |
| 31 | Transacciones y prepared statements | [31-transacciones-sql](31-transacciones-sql) | `Begin`/`Commit`/`Rollback`, `sql.Tx`, `Prepare` |
| 32 | Servidor HTTP básico (`net/http`) | [32-http-servidor](32-http-servidor) | Rutas con método y comodines, handlers, `http.ServeMux` |
| 33 | Cliente HTTP (`net/http.Client`) | [33-http-cliente](33-http-cliente) | Requests, timeouts, headers (contra un servidor local de prueba) |
| 34 | Middlewares HTTP | [34-http-middlewares](34-http-middlewares) | Logging, auth simple, recover, composición de handlers |
| 35 | Logging estructurado (`log/slog`) | [35-logging](35-logging) | Niveles, campos estructurados, JSON, `LogValuer` |
| 36 | Variables de entorno y configuración | [36-configuracion](36-configuracion) | `os.LookupEnv`, archivo `.env`, struct de config validado |
| 37 | Módulos, versionado y dependencias externas | [37-modulos-dependencias](37-modulos-dependencias) | `go get`, `go.sum`, semver, alias de import, `ReadBuildInfo` |
| 38 | Patrones de concurrencia (fan-in/fan-out, pipelines) | [38-patrones-concurrencia](38-patrones-concurrencia) | `errgroup`, cancelación coordinada, `SetLimit` |
| 39 | Validación de datos de entrada | [39-validacion](39-validacion) | Validación manual y con tags (`go-playground/validator`) |
| 40 | Serialización adicional (`encoding/xml`, `csv`) | [40-serializacion](40-serializacion) | Complementa el tema 19 (JSON) |
| 41 | Testing con mocks e interfaces | [41-testing-mocks](41-testing-mocks) | Inyección de dependencias para testear sin red ni base de datos |

Notas:
- Tema 32 (servidor HTTP): `go run ./32-http-servidor` queda escuchando en
  `http://localhost:8080`. Prueba las rutas con `curl` o el navegador desde
  otra terminal (los comandos están en el comentario inicial de `main.go`).
  Se detiene con Ctrl+C.
- Temas 33 y 34: levantan su propio servidor de prueba (`httptest`), así que
  no necesitan Internet ni otra terminal.
- Tema 36 (configuración): lee `36-configuracion/app.env`. Prueba a
  sobrescribir variables, por ejemplo en PowerShell:
  `$env:APP_PUERTO="9090"; go run ./36-configuracion`.
- Tema 41 (mocks): como en el tema 18, lo importante son los tests:
  `go test -v ./41-testing-mocks`.

---

## Índice: Go Avanzado

Temas de arquitectura, rendimiento y ecosistema para llevar proyectos Go a
producción.

| # | Tema | Carpeta | Notas |
|---|------|---------|-------|
| 42 | ORMs y migraciones (GORM, `golang-migrate`) | [42-orm-migraciones](42-orm-migraciones) | GORM en `main.go`; migraciones versionadas en `migrar/` |
| 43 | gRPC y Protocol Buffers | [43-grpc](43-grpc) | `.proto`, código generado, unario y streaming, metadata, deadlines |
| 44 | Autenticación y seguridad (JWT, bcrypt, TLS) | [44-seguridad](44-seguridad) | Hash de contraseñas, tokens firmados, HTTPS con certificado propio |
| 45 | WebSockets | [45-websockets](45-websockets) | Chat con hub, ping/pong, clientes simulados y página web |
| 46 | Profiling y benchmarking (`pprof`, `testing.B`) | [46-profiling-benchmarks](46-profiling-benchmarks) | `b.Loop()`, `-benchmem`, perfiles de CPU y memoria |
| 47 | Concurrencia avanzada (`sync`, `atomic`, worker pools) | [47-concurrencia-avanzada](47-concurrencia-avanzada) | Detector de carreras, `RWMutex`, `Pool`, `Cond`, semáforos |
| 48 | `go generate` y generación de código | [48-go-generate](48-go-generate) | `stringer` como herramienta del módulo y un generador propio |
| 49 | Observabilidad (métricas Prometheus, tracing OpenTelemetry) | [49-observabilidad](49-observabilidad) | Counter/Gauge/Histogram, spans propagados entre servicios |
| 50 | Arquitectura limpia / hexagonal en Go | [50-arquitectura-limpia](50-arquitectura-limpia) | Dominio, casos de uso, puertos y adaptadores intercambiables |
| 51 | Dockerización de aplicaciones Go | [51-docker](51-docker) | Multi-stage, distroless (~9 MB), healthcheck, apagado ordenado |
| 52 | CI/CD para proyectos Go | [52-ci-cd](52-ci-cd) | GitHub Actions, `golangci-lint`, GoReleaser, fuzzing, Makefile |
| 53 | Microservicios: comunicación y patrones | [53-microservicios](53-microservicios) | Service discovery, circuit breaker, reintentos, pub/sub con DLQ |

Notas:
- Tema 42: son dos programas, `go run ./42-orm-migraciones` (GORM) y
  `go run ./42-orm-migraciones/migrar -accion=demo` (migraciones). Están
  separados porque los dos drivers de SQLite registran el mismo nombre
  `"sqlite"` y no pueden convivir en un binario; además, en producción las
  migraciones suelen ejecutarse como un paso aparte. El segundo crea el
  archivo `42-orm-migraciones/biblioteca.db`.
- Tema 43: el código generado (`catalogopb/`) ya está incluido; solo hace
  falta `protoc` si modificas el `.proto` (ver el comentario de `main.go`).
- Tema 45: tras la simulación queda escuchando en `http://localhost:8081`;
  abre varias pestañas del navegador para chatear. Ctrl+C para salir.
- Tema 46: `go run` genera `cpu.prof` y `mem.prof`; los benchmarks se
  ejecutan con `go test -bench=. -benchmem ./46-profiling-benchmarks`.
- Tema 47: `go run -race ./47-concurrencia-avanzada -carrera` muestra el
  detector de carreras (requiere cgo: en Windows, gcc de MSYS2).
- Tema 48: `go generate ./48-go-generate` regenera `*_string.go` y
  `paises_gen.go` (ya incluidos).
- Tema 49: guarda las trazas completas en `49-observabilidad/trazas.json`.
- Tema 50: `go run ./50-arquitectura-limpia -repo=sqlite` usa otro adaptador
  sin cambiar la lógica; `go test ./50-arquitectura-limpia/...` ejecuta sus tests.
- Tema 51: requiere Docker. Desde la raíz:
  `docker build -f 51-docker/Dockerfile -t curso-go/servicio:1.0 .` y
  `docker run --rm -p 8084:8080 curso-go/servicio:1.0`. Sin Docker: `go run ./51-docker`.
- Tema 52: los workflows de ejemplo están en `52-ci-cd/.github/workflows/`;
  GitHub solo los ejecuta si se copian a `.github/workflows/` en la raíz del
  repositorio.

---

## Novedades de Go usadas

El proyecto aprovecha funciones recientes de Go. Si encuentras código
antiguo que hace lo mismo de otra forma, esta tabla ayuda a reconocerlo:

| Versión | Novedad | Dónde se ve |
|---------|---------|-------------|
| 1.21 | Paquetes `slices`, `maps` y `cmp`; funciones `min`/`max`; `log/slog` | 5, 6, 17, 22, 35 |
| 1.22 | `for i := range n`; variable nueva por iteración; rutas con método en `ServeMux` | 3, 14, 32 |
| 1.23 | Iteradores: `slices.Sorted(maps.Keys(m))`, `slices.Backward` | 6, 22, 34 |
| 1.24 | `testing.B.Loop`, `strings.SplitSeq`, `crypto/rand.Text`, directiva `tool` en go.mod | 36, 44, 46, 48 |
| 1.25 | `sync.WaitGroup.Go`; GOMAXPROCS según el límite de CPU del contenedor | 14, 20, 38, 45, 47, 51 |
| 1.26 | `new(expresión)`, `errors.AsType`, iteradores `reflect.Type.Methods()` | 9, 12, 29, 33, 39, 40 |
| 1.27 | Métodos genéricos; campos promovidos en literales de struct; `encoding/json/v2`; paquete `uuid` | 7, 17, 19, 37 |

## Dependencias

Las dependencias externas están en `go.mod` con sus versiones fijadas, y
`go.sum` guarda sus hashes para que todos compilen exactamente el mismo
código. Para actualizarlas:

```
go get -u ./...     actualiza las dependencias (versiones menores y parches)
go get -u tool      actualiza las herramientas declaradas en go.mod (stringer)
go mod tidy         limpia go.mod y go.sum
go build ./... && go vet ./... && go test ./...   verificar que todo sigue funcionando
```

Al actualizar Go, cambia también la imagen base del `Dockerfile` del tema 51
(`golang:1.27-alpine`) para que coincida con la directiva `go` de `go.mod`.
