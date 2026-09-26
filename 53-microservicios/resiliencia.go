package main

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"time"
)

// ==================== CIRCUIT BREAKER ====================
// Si un servicio está caído, seguir llamándolo empeora todo: cada request
// espera el timeout, se acumulan goroutines y el servicio caído no puede
// recuperarse bajo la carga. El circuit breaker (como un disyuntor eléctrico):
//
//   CERRADO  --(N fallos seguidos)-->  ABIERTO  --(pasa el tiempo de espera)-->  SEMI-ABIERTO
//      ^                                  ^                                            |
//      |_______(la llamada de prueba sale bien)____|___(la prueba falla)______________|
//
//   Cerrado: las llamadas pasan normalmente.
//   Abierto: se rechazan al instante SIN llamar (falla rápido).
//   Semi-abierto: se deja pasar UNA llamada de prueba.
// Librerías reales: github.com/sony/gobreaker, failsafe-go.

type EstadoCircuito int

const (
	Cerrado EstadoCircuito = iota
	Abierto
	SemiAbierto
)

func (e EstadoCircuito) String() string {
	return [...]string{"CERRADO", "ABIERTO", "SEMI-ABIERTO"}[e]
}

var ErrCircuitoAbierto = errors.New("circuito abierto")

type CircuitBreaker struct {
	mu            sync.Mutex
	nombre        string
	estado        EstadoCircuito
	fallos        int
	umbralFallos  int
	espera        time.Duration
	abiertoDesde  time.Time
	pruebaEnCurso bool
	alCambiar     func(nombre string, de, a EstadoCircuito)
}

func NuevoCircuitBreaker(nombre string, umbral int, espera time.Duration, alCambiar func(string, EstadoCircuito, EstadoCircuito)) *CircuitBreaker {
	return &CircuitBreaker{nombre: nombre, umbralFallos: umbral, espera: espera, alCambiar: alCambiar}
}

func (cb *CircuitBreaker) cambiar(a EstadoCircuito) {
	if cb.estado != a && cb.alCambiar != nil {
		cb.alCambiar(cb.nombre, cb.estado, a)
	}
	cb.estado = a
}

// Ejecutar corre f protegida por el breaker
func (cb *CircuitBreaker) Ejecutar(f func() error) error {
	cb.mu.Lock()
	if cb.estado == Abierto {
		if time.Since(cb.abiertoDesde) < cb.espera {
			cb.mu.Unlock()
			return ErrCircuitoAbierto // fallar rápido, sin llamar
		}
		cb.cambiar(SemiAbierto)
	}
	if cb.estado == SemiAbierto {
		if cb.pruebaEnCurso {
			cb.mu.Unlock()
			return ErrCircuitoAbierto // solo una llamada de prueba a la vez
		}
		cb.pruebaEnCurso = true
	}
	cb.mu.Unlock()

	err := f() // la llamada real, sin tener el lock tomado

	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.pruebaEnCurso = false
	if err != nil {
		cb.fallos++
		if cb.estado == SemiAbierto || cb.fallos >= cb.umbralFallos {
			cb.cambiar(Abierto)
			cb.abiertoDesde = time.Now()
		}
		return err
	}
	cb.fallos = 0
	cb.cambiar(Cerrado)
	return nil
}

func (cb *CircuitBreaker) Estado() EstadoCircuito {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.estado
}

// ==================== REINTENTOS CON BACKOFF EXPONENCIAL ====================
// Reintentar ayuda con fallos transitorios, pero mal hecho causa una
// "estampida": todos los clientes reintentan a la vez. Por eso:
//   - backoff exponencial: esperar 50ms, 100ms, 200ms... entre intentos
//   - jitter: sumar aleatoriedad para que los clientes no se sincronicen
//   - límite de intentos y respetar el context (deadline del request original)
//   - reintentar SOLO operaciones idempotentes (GET, o POST con clave de idempotencia)

// ErrNoReintentable marca errores que no tiene sentido reintentar (p. ej. un 404)
type ErrNoReintentable struct{ Err error }

func (e ErrNoReintentable) Error() string { return e.Err.Error() }
func (e ErrNoReintentable) Unwrap() error { return e.Err }

func Reintentar(ctx context.Context, intentos int, base time.Duration, f func(intento int) error) error {
	var err error
	for i := range intentos {
		if err = f(i + 1); err == nil {
			return nil
		}
		var noReintentable ErrNoReintentable
		if errors.As(err, &noReintentable) || i == intentos-1 {
			break
		}
		// backoff = base * 2^i, más jitter de hasta el 50%
		espera := base << i
		espera += time.Duration(rand.Int64N(int64(espera) / 2))
		select {
		case <-time.After(espera):
		case <-ctx.Done():
			return errors.Join(err, ctx.Err())
		}
	}
	return err
}
