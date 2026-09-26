// Tema 39: Validación de datos de entrada
// Todo dato que llega desde afuera (HTTP, archivos, CLI) debe validarse antes
// de usarse. Dos enfoques:
//  1. Validación manual: un método Validar() que devuelve errores claros.
//     Sin dependencias, explícito, fácil de testear.
//  2. Validación declarativa con tags de struct usando
//     github.com/go-playground/validator/v10 (la librería más usada; la
//     integran frameworks como Gin). Se basa en reflection (tema 29).
//
// Ejecutar con: go run ./39-validacion
package main

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
)

// ================= 1) VALIDACIÓN MANUAL =================

// ErrorCampo describe un problema en un campo concreto
type ErrorCampo struct {
	Campo   string
	Mensaje string
}

func (e ErrorCampo) Error() string { return e.Campo + ": " + e.Mensaje }

// ErroresValidacion agrupa todos los problemas encontrados, para informar
// todo de una vez en vez de obligar al usuario a corregir de a uno.
type ErroresValidacion []ErrorCampo

func (es ErroresValidacion) Error() string {
	partes := make([]string, len(es))
	for i, e := range es {
		partes[i] = e.Error()
	}
	return strings.Join(partes, "; ")
}

type Registro struct {
	Nombre   string
	Email    string
	Edad     int
	Password string
	Rol      string
}

func (r Registro) Validar() error {
	var errs ErroresValidacion
	agregar := func(campo, msg string) { errs = append(errs, ErrorCampo{campo, msg}) }

	nombre := strings.TrimSpace(r.Nombre)
	switch {
	case nombre == "":
		agregar("nombre", "es obligatorio")
	case utf8.RuneCountInString(nombre) < 2: // contar runas, no bytes (tema 15)
		agregar("nombre", "debe tener al menos 2 caracteres")
	}

	if _, err := mail.ParseAddress(r.Email); err != nil {
		agregar("email", "no es una dirección válida")
	}

	if r.Edad < 18 || r.Edad > 120 {
		agregar("edad", "debe estar entre 18 y 120")
	}

	if len(r.Password) < 8 || !strings.ContainsAny(r.Password, "0123456789") {
		agregar("password", "mínimo 8 caracteres e incluir un número")
	}

	rolesValidos := map[string]bool{"admin": true, "editor": true, "lector": true}
	if !rolesValidos[r.Rol] {
		agregar("rol", fmt.Sprintf("%q no es un rol válido", r.Rol))
	}

	if len(errs) == 0 {
		return nil // IMPORTANTE: devolver nil literal, no un ErroresValidacion vacío
	}
	return errs
}

// ============ 2) VALIDACIÓN CON TAGS (validator) ============

type Direccion struct {
	Calle  string `validate:"required"`
	Ciudad string `validate:"required"`
	CP     string `validate:"required,numeric,len=5"`
}

type Pedido struct {
	ClienteEmail string            `validate:"required,email"`
	Cantidad     int               `validate:"gte=1,lte=100"`
	Moneda       string            `validate:"oneof=ARS USD EUR"`
	Cupon        string            `validate:"omitempty,alphanum,len=8"` // opcional, pero si viene se valida
	Envio        Direccion         `validate:"required"`                 // valida el struct anidado
	Etiquetas    []string          `validate:"max=3,dive,min=2"`         // dive: valida cada elemento
	Notas        map[string]string `validate:"omitempty"`
	FechaEntrega string            `validate:"required,datetime=2006-01-02"`
	SKU          string            `validate:"required,sku"` // regla personalizada (ver abajo)
}

// validarSKU es una regla propia: formato "ABC-1234"
func validarSKU(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	letras, numeros, ok := strings.Cut(s, "-")
	if !ok || len(letras) != 3 || len(numeros) != 4 {
		return false
	}
	return strings.ToUpper(letras) == letras && strings.Trim(numeros, "0123456789") == ""
}

// mensajeAmigable traduce cada fallo del validator a un texto en español
func mensajeAmigable(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "es obligatorio"
	case "email":
		return "debe ser un email válido"
	case "gte":
		return "debe ser mayor o igual a " + fe.Param()
	case "lte":
		return "debe ser menor o igual a " + fe.Param()
	case "oneof":
		return "debe ser uno de: " + fe.Param()
	case "len":
		return "debe tener longitud " + fe.Param()
	case "max":
		return "admite como máximo " + fe.Param() + " elementos"
	case "min":
		return "debe tener al menos " + fe.Param() + " caracteres"
	case "datetime":
		return "debe tener formato AAAA-MM-DD"
	case "sku":
		return "debe tener formato ABC-1234"
	default:
		return "no cumple la regla " + fe.Tag()
	}
}

func mostrarErroresValidator(err error) {
	// El validator devuelve validator.ValidationErrors (un slice de FieldError)
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		fmt.Println("   error inesperado:", err)
		return
	}
	for _, fe := range verrs {
		// Namespace incluye la ruta completa, p. ej. Pedido.Envio.CP
		fmt.Printf("   - %-22s %s (valor: %v)\n", fe.Namespace(), mensajeAmigable(fe), fe.Value())
	}
}

func main() {
	fmt.Println("===== 1) Validación manual =====")
	valido := Registro{Nombre: "Ana", Email: "ana@ejemplo.com", Edad: 30, Password: "segura123", Rol: "editor"}
	fmt.Println("Registro válido ->", valido.Validar())

	invalido := Registro{Nombre: " A ", Email: "ana@", Edad: 15, Password: "corta", Rol: "jefe"}
	err := invalido.Validar()
	fmt.Println("Registro inválido:")
	// errors.AsType (Go 1.26+, ver tema 12) recupera el tipo concreto para
	// poder recorrer cada error de campo
	if errsVal, ok := errors.AsType[ErroresValidacion](err); ok {
		for _, e := range errsVal {
			fmt.Printf("   - %-9s %s\n", e.Campo, e.Mensaje)
		}
	}

	fmt.Println("\n===== 2) Validación con tags (go-playground/validator) =====")
	// Crear el validador UNA vez y reutilizarlo: cachea la info de los structs.
	// WithRequiredStructEnabled activa el comportamiento que será el default
	// en la próxima versión mayor (required en structs anidados).
	validar := validator.New(validator.WithRequiredStructEnabled())
	if err := validar.RegisterValidation("sku", validarSKU); err != nil {
		panic(err)
	}

	pedidoOK := Pedido{
		ClienteEmail: "bruno@ejemplo.com",
		Cantidad:     3,
		Moneda:       "USD",
		Envio:        Direccion{Calle: "Av. Siempre Viva 742", Ciudad: "Springfield", CP: "12345"},
		Etiquetas:    []string{"regalo", "urgente"},
		FechaEntrega: "2026-10-15",
		SKU:          "TEC-0042",
	}
	fmt.Println("Pedido válido ->", validar.Struct(pedidoOK))

	pedidoMalo := Pedido{
		ClienteEmail: "no-es-email",
		Cantidad:     0,
		Moneda:       "BTC",
		Cupon:        "abc",
		Envio:        Direccion{Calle: "Falsa 123", CP: "12A"},
		Etiquetas:    []string{"ok", "x", "tres", "cuatro"},
		FechaEntrega: "15/10/2026",
		SKU:          "tec-42",
	}
	fmt.Println("Pedido inválido:")
	mostrarErroresValidator(validar.Struct(pedidoMalo))

	// También se pueden validar variables sueltas
	fmt.Println("\nValidar una variable suelta:")
	fmt.Println("   'https://go.dev' es url ->", validar.Var("https://go.dev", "url") == nil)
	fmt.Println("   'hola' es url           ->", validar.Var("hola", "url") == nil)
}
