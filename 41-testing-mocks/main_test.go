package main

import (
	"errors"
	"testing"
	"time"
)

// ---------- Dobles de prueba ----------
// Terminología habitual:
//   - Stub: devuelve respuestas predefinidas.
//   - Fake: implementación funcional pero simplificada (p. ej. un map en vez de DB).
//   - Mock/Spy: además registra CÓMO fue llamado, para verificarlo después.
// En Go se escriben a mano en pocas líneas gracias a las interfaces implícitas.

// repoFalso es un fake con "ganchos" para forzar errores
type repoFalso struct {
	datos        map[string]Usuario
	errBuscar    error // si no es nil, BuscarPorEmail lo devuelve
	errGuardar   error
	llamadasGuar int
}

func nuevoRepoFalso() *repoFalso {
	return &repoFalso{datos: map[string]Usuario{}}
}

func (r *repoFalso) BuscarPorEmail(email string) (Usuario, error) {
	if r.errBuscar != nil {
		return Usuario{}, r.errBuscar
	}
	u, ok := r.datos[email]
	if !ok {
		return Usuario{}, ErrNoEncontrado
	}
	return u, nil
}

func (r *repoFalso) Guardar(u Usuario) error {
	r.llamadasGuar++
	if r.errGuardar != nil {
		return r.errGuardar
	}
	r.datos[u.Email] = u
	return nil
}

// notificadorEspia registra cada mensaje enviado (spy)
type mensaje struct{ destino, asunto string }

type notificadorEspia struct {
	enviados []mensaje
	err      error
}

func (n *notificadorEspia) Enviar(destino, asunto, cuerpo string) error {
	n.enviados = append(n.enviados, mensaje{destino, asunto})
	return n.err
}

// relojFijo siempre devuelve la misma hora: tests deterministas
type relojFijo struct{ t time.Time }

func (r relojFijo) Ahora() time.Time { return r.t }

var horaFija = time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

// Verificación en tiempo de compilación de que los dobles cumplen las
// interfaces: si falta un método, el test ni compila.
var (
	_ RepositorioUsuarios = (*repoFalso)(nil)
	_ Notificador         = (*notificadorEspia)(nil)
	_ Reloj               = relojFijo{}
)

// ---------- Tests ----------

func TestRegistrar_Exitoso(t *testing.T) {
	repo := nuevoRepoFalso()
	notif := &notificadorEspia{}
	svc := NuevoServicioRegistro(repo, notif, relojFijo{horaFija})

	u, err := svc.Registrar("  Ana@Ejemplo.COM ", "Ana")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	// Verificar el resultado
	if u.Email != "ana@ejemplo.com" {
		t.Errorf("email = %q; se esperaba normalizado a minúsculas", u.Email)
	}
	if !u.CreadoEn.Equal(horaFija) {
		t.Errorf("CreadoEn = %v; se esperaba %v", u.CreadoEn, horaFija)
	}
	// Verificar el efecto sobre el repositorio
	if _, ok := repo.datos["ana@ejemplo.com"]; !ok {
		t.Error("el usuario no se guardó en el repositorio")
	}
	// Verificar la interacción con el notificador (lo que hace un mock)
	if len(notif.enviados) != 1 {
		t.Fatalf("se enviaron %d emails; se esperaba 1", len(notif.enviados))
	}
	if got := notif.enviados[0]; got.destino != "ana@ejemplo.com" || got.asunto != "¡Bienvenido/a, Ana!" {
		t.Errorf("email enviado = %+v", got)
	}
}

func TestRegistrar_Errores(t *testing.T) {
	errDB := errors.New("conexión perdida")

	casos := []struct {
		nombre      string
		email       string
		preparar    func(r *repoFalso) // configura el fake para cada caso
		errEsperado error
		debeGuardar bool
	}{
		{
			nombre:      "datos inválidos",
			email:       "sin-arroba",
			errEsperado: ErrDatos,
		},
		{
			nombre: "email duplicado",
			email:  "ana@ejemplo.com",
			preparar: func(r *repoFalso) {
				r.datos["ana@ejemplo.com"] = Usuario{Email: "ana@ejemplo.com"}
			},
			errEsperado: ErrYaExiste,
		},
		{
			nombre:      "falla la búsqueda en la DB",
			email:       "ana@ejemplo.com",
			preparar:    func(r *repoFalso) { r.errBuscar = errDB },
			errEsperado: errDB, // se envuelve con %w, así que errors.Is lo encuentra
		},
		{
			nombre:      "falla el guardado",
			email:       "ana@ejemplo.com",
			preparar:    func(r *repoFalso) { r.errGuardar = errDB },
			errEsperado: errDB,
			debeGuardar: true,
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			repo := nuevoRepoFalso()
			if c.preparar != nil {
				c.preparar(repo)
			}
			notif := &notificadorEspia{}
			svc := NuevoServicioRegistro(repo, notif, relojFijo{horaFija})

			_, err := svc.Registrar(c.email, "Ana")

			if !errors.Is(err, c.errEsperado) {
				t.Errorf("error = %v; se esperaba %v", err, c.errEsperado)
			}
			if intentoGuardar := repo.llamadasGuar > 0; intentoGuardar != c.debeGuardar {
				t.Errorf("¿intentó guardar? %v; se esperaba %v", intentoGuardar, c.debeGuardar)
			}
			if len(notif.enviados) != 0 {
				t.Errorf("no debería enviarse email si el registro falla; se enviaron %d", len(notif.enviados))
			}
		})
	}
}

func TestRegistrar_FallaEmailNoRevierteRegistro(t *testing.T) {
	repo := nuevoRepoFalso()
	notif := &notificadorEspia{err: errors.New("SMTP caído")}
	svc := NuevoServicioRegistro(repo, notif, relojFijo{horaFija})

	_, err := svc.Registrar("bruno@ejemplo.com", "Bruno")
	if err != nil {
		t.Fatalf("el fallo del email no debería fallar el registro: %v", err)
	}
	if _, ok := repo.datos["bruno@ejemplo.com"]; !ok {
		t.Error("el usuario debería haberse guardado igual")
	}
	if len(notif.enviados) != 1 {
		t.Errorf("se esperaba 1 intento de envío, hubo %d", len(notif.enviados))
	}
}
