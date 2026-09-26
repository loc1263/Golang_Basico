# Tema 52: Makefile con los mismos pasos que el CI, para ejecutarlos local.
# Uso (desde la raíz del proyecto):  make -f 52-ci-cd/Makefile ci
# En Windows, make viene con MSYS2/Git Bash/WSL (o: choco install make).
# Nota: las líneas de las recetas DEBEN empezar con TAB, no espacios.

PKG      := ./52-ci-cd
BINARIO  := contador
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null)
LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: ci fmt vet lint test cover vuln build build-all clean

## ci: todo lo que corre en el pipeline
ci: fmt vet lint test

fmt:
	@test -z "$$(gofmt -l $(PKG))" || (echo "Ejecuta: gofmt -w $(PKG)"; gofmt -l $(PKG); exit 1)

vet:
	go vet $(PKG)/...

lint:
	golangci-lint run --config=52-ci-cd/.golangci.yml $(PKG)/...

test:
	go test -race -count=1 $(PKG)/...

cover:
	go test -coverprofile=coverage.out $(PKG)/...
	go tool cover -html=coverage.out   # abre el reporte en el navegador

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest $(PKG)/...

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARIO) $(PKG)

## build-all: compilación cruzada para varias plataformas
build-all:
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARIO)-linux-amd64 $(PKG)
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARIO)-linux-arm64 $(PKG)
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARIO)-darwin-arm64 $(PKG)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARIO)-windows-amd64.exe $(PKG)

clean:
	rm -rf bin dist coverage.out
