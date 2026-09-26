// Tema 36: Variables de entorno y configuración
// Una práctica común (12-factor apps) es leer la configuración desde variables
// de entorno, con valores por defecto razonables, y cargarla UNA vez al inicio
// en un struct tipado que se pasa al resto del programa.
// También es habitual un archivo .env para desarrollo local: aquí lo leemos a
// mano para entender cómo funciona (en proyectos reales suele usarse
// github.com/joho/godotenv).
//
// Ejecutar con: go run ./36-configuracion
// Probar sobreescribiendo variables (PowerShell):
//
//	$env:APP_PUERTO="9090"; $env:APP_DEBUG="true"; go run ./36-configuracion
//
// (bash): APP_PUERTO=9090 APP_DEBUG=true go run ./36-configuracion
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Entorno     string
	Puerto      int
	Debug       bool
	TimeoutHTTP time.Duration
	DatabaseURL string
	Origenes    []string
}

// cargarDotEnv lee un archivo con líneas CLAVE=valor y las define como
// variables de entorno. Las variables que YA existen tienen prioridad, así
// el entorno real siempre gana sobre el archivo.
func cargarDotEnv(ruta string) error {
	f, err := os.Open(ruta)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		linea := strings.TrimSpace(sc.Text())
		if linea == "" || strings.HasPrefix(linea, "#") {
			continue // líneas vacías y comentarios
		}
		clave, valor, ok := strings.Cut(linea, "=")
		if !ok {
			return fmt.Errorf("%s:%d: se esperaba CLAVE=valor", ruta, n)
		}
		clave = strings.TrimSpace(clave)
		valor = strings.Trim(strings.TrimSpace(valor), `"'`) // quitar comillas
		if _, existe := os.LookupEnv(clave); !existe {
			os.Setenv(clave, valor)
		}
	}
	return sc.Err()
}

// Funciones auxiliares: leer con valor por defecto y convertir el tipo.
// os.Getenv devuelve "" tanto si la variable no existe como si está vacía;
// os.LookupEnv permite distinguir ambos casos.

func texto(clave, defecto string) string {
	if v, ok := os.LookupEnv(clave); ok && v != "" {
		return v
	}
	return defecto
}

// Los errores de conversión se acumulan en vez de cortar en el primero,
// para mostrar al usuario todos los problemas de configuración de una vez.
type lector struct{ errs []error }

func (l *lector) entero(clave string, defecto int) int {
	v, ok := os.LookupEnv(clave)
	if !ok {
		return defecto
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s=%q no es un entero", clave, v))
		return defecto
	}
	return n
}

func (l *lector) booleano(clave string, defecto bool) bool {
	v, ok := os.LookupEnv(clave)
	if !ok {
		return defecto
	}
	b, err := strconv.ParseBool(v) // acepta 1, t, true, TRUE, 0, f, false...
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s=%q no es un booleano", clave, v))
		return defecto
	}
	return b
}

func (l *lector) duracion(clave string, defecto time.Duration) time.Duration {
	v, ok := os.LookupEnv(clave)
	if !ok {
		return defecto
	}
	d, err := time.ParseDuration(v) // "5s", "1m30s", "250ms"
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s=%q no es una duración", clave, v))
		return defecto
	}
	return d
}

func (l *lector) lista(clave string, defecto []string) []string {
	v, ok := os.LookupEnv(clave)
	if !ok || v == "" {
		return defecto
	}
	partes := strings.Split(v, ",")
	for i := range partes {
		partes[i] = strings.TrimSpace(partes[i])
	}
	return partes
}

// Cargar arma el Config y valida las reglas de negocio
func Cargar() (Config, error) {
	var l lector
	cfg := Config{
		Entorno:     texto("APP_ENTORNO", "desarrollo"),
		Puerto:      l.entero("APP_PUERTO", 8080),
		Debug:       l.booleano("APP_DEBUG", false),
		TimeoutHTTP: l.duracion("APP_TIMEOUT_HTTP", 10*time.Second),
		DatabaseURL: texto("DATABASE_URL", ""),
		Origenes:    l.lista("APP_ORIGENES", []string{"http://localhost:3000"}),
	}

	if cfg.Puerto < 1 || cfg.Puerto > 65535 {
		l.errs = append(l.errs, fmt.Errorf("APP_PUERTO=%d fuera de rango", cfg.Puerto))
	}
	if cfg.Entorno == "produccion" && cfg.DatabaseURL == "" {
		l.errs = append(l.errs, errors.New("DATABASE_URL es obligatoria en producción"))
	}
	// errors.Join combina varios errores en uno (devuelve nil si no hay ninguno)
	return cfg, errors.Join(l.errs...)
}

// String oculta datos sensibles al imprimir la configuración
func (c Config) String() string {
	db := "(vacía)"
	if c.DatabaseURL != "" {
		db = "****"
	}
	return fmt.Sprintf("entorno=%s puerto=%d debug=%v timeout=%v db=%s origenes=%v",
		c.Entorno, c.Puerto, c.Debug, c.TimeoutHTTP, db, c.Origenes)
}

func main() {
	// El archivo está junto a este main.go. La ruta es relativa al directorio
	// desde donde se ejecuta el programa (la raíz del proyecto).
	if err := cargarDotEnv("36-configuracion/app.env"); err != nil {
		fmt.Println("aviso: no se cargó app.env:", err)
	}

	fmt.Println("1) Configuración cargada (defaults + app.env + entorno):")
	cfg, err := Cargar()
	if err != nil {
		fmt.Println("configuración inválida:\n" + err.Error())
		os.Exit(1)
	}
	fmt.Println("  ", cfg)

	// Simular una configuración con varios errores a la vez
	fmt.Println("\n2) Configuración inválida (errores acumulados):")
	os.Setenv("APP_ENTORNO", "produccion")
	os.Setenv("APP_PUERTO", "99999")
	os.Setenv("APP_DEBUG", "quizas")
	os.Unsetenv("DATABASE_URL")
	if _, err := Cargar(); err != nil {
		// errors.Join separa cada error con "\n". strings.SplitSeq (Go 1.24+)
		// recorre las partes sin crear un slice intermedio.
		for linea := range strings.SplitSeq(err.Error(), "\n") {
			fmt.Println("   -", linea)
		}
	}

	// Listar variables de entorno del proceso con un prefijo
	fmt.Println("\n3) Variables APP_* del proceso:")
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "APP_") {
			fmt.Println("  ", kv)
		}
	}
}
