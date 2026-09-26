# syntax=docker/dockerfile:1
# Tema 51: build multi-etapa para una aplicación Go.
# Construir desde la raíz del proyecto:
#   docker build -f 51-docker/Dockerfile -t curso-go/servicio:1.0 --build-arg VERSION=1.0 .

# ======================= Etapa 1: compilación =======================
# Imagen con el toolchain completo de Go (~300 MB). Solo se usa para compilar.
# La versión coincide con la directiva `go` de go.mod.
FROM golang:1.27-alpine AS build

WORKDIR /src

# Patrón clásico que verás en muchos proyectos: copiar primero go.mod/go.sum
# y ejecutar `go mod download` en una capa propia, para que Docker la cachee
# mientras las dependencias no cambien:
#     COPY go.mod go.sum ./
#     RUN go mod download
# Aquí usamos cache mounts (abajo), que logran lo mismo y además solo
# descargan los módulos que este paquete importa (este go.mod tiene muchos).

# Copiar el código (Dockerfile.dockerignore excluye lo innecesario)
COPY . .

ARG VERSION=dev
# TARGETOS/TARGETARCH los define BuildKit: permite construir imágenes para
# otras plataformas (docker buildx build --platform linux/arm64 ...)
ARG TARGETOS TARGETARCH

# Compilar un binario estático:
#    CGO_ENABLED=0    sin dependencias de C -> funciona en imágenes sin libc
#    -trimpath        quita rutas locales del binario (builds reproducibles)
#    -ldflags "-s -w" quita símbolos de depuración (binario más chico)
#    -X main.version  inyecta la versión en la variable `version`
#    Las cache mounts reutilizan módulos y compilaciones entre builds, y solo
#    se descargan los módulos que este paquete realmente importa.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/servicio ./51-docker

# ======================= Etapa 2: ejecución =======================
# distroless/static: solo certificados CA, zona horaria y /etc/passwd.
# Sin shell ni gestor de paquetes: nada que un atacante pueda aprovechar.
# Alternativas: `scratch` (vacía, aún más chica) o `alpine` (si necesitas shell para depurar).
FROM gcr.io/distroless/static-debian12:nonroot

# Solo el binario pasa a la imagen final: el código fuente y el toolchain se descartan
COPY --from=build /out/servicio /servicio

# Ejecutar como usuario sin privilegios (el tag :nonroot ya lo define; lo explicitamos)
USER nonroot:nonroot

ENV PORT=8080
EXPOSE 8080

# El propio binario hace de cliente para el healthcheck (no hay curl en distroless)
HEALTHCHECK --interval=10s --timeout=3s --start-period=2s --retries=3 \
    CMD ["/servicio", "-healthcheck"]

# Forma "exec" (JSON): el proceso es PID 1 y recibe SIGTERM directamente.
# Con la forma shell (ENTRYPOINT /servicio) la señal la recibiría el shell.
ENTRYPOINT ["/servicio"]
