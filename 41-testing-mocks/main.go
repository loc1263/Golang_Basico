// Tema 41: Testing con mocks e interfaces
// Para testear lógica que depende de una base de datos, una API externa, el
// reloj o el envío de emails, NO conviene usar los servicios reales en los
// tests (lentos, frágiles, con efectos secundarios). La solución idiomática:
//  1. La lógica depende de INTERFACES pequeñas, no de tipos concretos.
//  2. Las dependencias se reciben desde afuera (inyección de dependencias),
//     normalmente en un constructor.
//  3. En producción se pasan implementaciones reales; en los tests,
//     implementaciones falsas (fakes/stubs/mocks) escritas a mano.
//
// Este archivo contiene el código "de producción"; las pruebas están en
// main_test.go.
// Ejecutar el programa:  go run ./41-testing-mocks
// Ejecutar los tests:    go test -v ./41-testing-mocks
package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ---------- Dependencias expresadas como interfaces ----------
// "Acepta interfaces, devuelve structs": las interfaces se definen donde se
// USAN (aquí), con solo los métodos que el servicio necesita.

type RepositorioUsuarios interface {
	BuscarPorEmail(email string) (Usuario, error)
	Guardar(u Usuario) error
}

type Notificador interface {
	Enviar(destino, asunto, cuerpo string) error
}

// Reloj abstrae time.Now para poder fijar la hora en los tests
type Reloj interface {
	Ahora() time.Time
}

// ---------- Modelo y errores ----------

type Usuario struct {
	Email    string
	Nombre   string
	CreadoEn time.Time
}

var (
	ErrNoEncontrado = errors.New("usuario no encontrado")
	ErrYaExiste     = errors.New("el email ya está registrado")
	ErrDatos        = errors.New("datos inválidos")
)

// ---------- Lógica de negocio ----------

type ServicioRegistro struct {
	repo  RepositorioUsuarios
	notif Notificador
	reloj Reloj
}

// NuevoServicioRegistro recibe las dependencias: aquí ocurre la inyección
func NuevoServicioRegistro(repo RepositorioUsuarios, notif Notificador, reloj Reloj) *ServicioRegistro {
	return &ServicioRegistro{repo: repo, notif: notif, reloj: reloj}
}

// Registrar contiene la lógica que queremos testear:
//   - valida datos
//   - rechaza emails duplicados
//   - guarda el usuario con la fecha actual
//   - envía un email de bienvenida (si falla, el registro NO se revierte)
func (s *ServicioRegistro) Registrar(email, nombre string) (Usuario, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || strings.TrimSpace(nombre) == "" {
		return Usuario{}, ErrDatos
	}

	_, err := s.repo.BuscarPorEmail(email)
	switch {
	case err == nil:
		return Usuario{}, ErrYaExiste
	case !errors.Is(err, ErrNoEncontrado):
		return Usuario{}, fmt.Errorf("buscar usuario: %w", err)
	}

	u := Usuario{Email: email, Nombre: nombre, CreadoEn: s.reloj.Ahora()}
	if err := s.repo.Guardar(u); err != nil {
		return Usuario{}, fmt.Errorf("guardar usuario: %w", err)
	}

	asunto := "¡Bienvenido/a, " + nombre + "!"
	if err := s.notif.Enviar(email, asunto, "Gracias por registrarte."); err != nil {
		// Decisión de negocio: el email es "best effort", no se falla el registro
		fmt.Println("aviso: no se pudo enviar el email:", err)
	}
	return u, nil
}

// ---------- Implementaciones "reales" (simplificadas) ----------

type repoMemoria struct{ datos map[string]Usuario }

func (r *repoMemoria) BuscarPorEmail(email string) (Usuario, error) {
	u, ok := r.datos[email]
	if !ok {
		return Usuario{}, ErrNoEncontrado
	}
	return u, nil
}

func (r *repoMemoria) Guardar(u Usuario) error {
	r.datos[u.Email] = u
	return nil
}

// notificadorConsola simula un servicio SMTP imprimiendo en pantalla
type notificadorConsola struct{}

func (notificadorConsola) Enviar(destino, asunto, cuerpo string) error {
	fmt.Printf("  [email] para=%s asunto=%q\n", destino, asunto)
	return nil
}

type relojSistema struct{}

func (relojSistema) Ahora() time.Time { return time.Now() }

func main() {
	// En producción se "cablean" las implementaciones reales
	svc := NuevoServicioRegistro(
		&repoMemoria{datos: map[string]Usuario{}},
		notificadorConsola{},
		relojSistema{},
	)

	for _, intento := range []struct{ email, nombre string }{
		{"Ana@Ejemplo.com", "Ana"},
		{"ana@ejemplo.com", "Ana otra vez"},
		{"sin-arroba", "Bruno"},
	} {
		u, err := svc.Registrar(intento.email, intento.nombre)
		if err != nil {
			fmt.Printf("registrar %q: %v\n", intento.email, err)
			continue
		}
		fmt.Printf("registrado: %s (%s)\n", u.Nombre, u.Email)
	}
	fmt.Println("\nCorre `go test -v ./41-testing-mocks` para ver los tests con dobles de prueba.")
}
