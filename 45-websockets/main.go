// Tema 45: WebSockets
// HTTP es petición-respuesta: el servidor no puede enviar datos cuando quiere.
// WebSocket empieza como un request HTTP normal con "Upgrade: websocket" y,
// si el servidor acepta, la conexión TCP queda abierta y ambos lados pueden
// enviarse mensajes en cualquier momento (chats, notificaciones, juegos,
// dashboards en tiempo real).
//
// Usamos github.com/gorilla/websocket, la librería más difundida.
// Patrón clásico de chat: un "hub" central y, por cada cliente, una goroutine
// que lee y otra que escribe (una conexión admite a lo sumo un lector y un
// escritor concurrentes).
//
// Ejecutar con: go run ./45-websockets
//
//	El programa simula 3 clientes automáticamente y luego queda escuchando:
//	abre http://localhost:8081 en varias pestañas del navegador para chatear.
//	Detener con Ctrl+C.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	tiempoEscritura = 10 * time.Second
	tiempoPong      = 60 * time.Second
	intervaloPing   = tiempoPong * 9 / 10 // ping antes de que venza el plazo del pong
	tamanoMaxMsg    = 512
)

type Mensaje struct {
	De    string    `json:"de"`
	Texto string    `json:"texto"`
	Hora  time.Time `json:"hora"`
}

// ======================= HUB =======================

// Hub mantiene los clientes conectados y reenvía cada mensaje a todos.
// Toda modificación del mapa ocurre en la goroutine de run(): sin mutex,
// usando channels ("no comuniques compartiendo memoria...").
type Hub struct {
	clientes  map[*Cliente]bool
	difundir  chan Mensaje
	registrar chan *Cliente
	baja      chan *Cliente
}

func nuevoHub() *Hub {
	return &Hub{
		clientes:  make(map[*Cliente]bool),
		difundir:  make(chan Mensaje),
		registrar: make(chan *Cliente),
		baja:      make(chan *Cliente),
	}
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.registrar:
			h.clientes[c] = true
		case c := <-h.baja:
			if h.clientes[c] {
				delete(h.clientes, c)
				close(c.enviar) // hace terminar a su goroutine de escritura
			}
		case m := <-h.difundir:
			for c := range h.clientes {
				select {
				case c.enviar <- m:
				default:
					// Cliente lento con el buffer lleno: se lo desconecta para
					// no frenar a todos los demás
					delete(h.clientes, c)
					close(c.enviar)
				}
			}
		}
	}
}

// ======================= CLIENTE (lado servidor) =======================

type Cliente struct {
	hub    *Hub
	conn   *websocket.Conn
	nombre string
	enviar chan Mensaje // buffer de salida
}

// leer: una goroutine por cliente, la ÚNICA que llama a conn.ReadMessage
func (c *Cliente) leer() {
	defer func() {
		c.hub.baja <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(tamanoMaxMsg)
	c.conn.SetReadDeadline(time.Now().Add(tiempoPong))
	// Cada pong recibido extiende el plazo: detecta conexiones muertas
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(tiempoPong))
	})

	for {
		_, datos, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Println("error de lectura:", err)
			}
			return
		}
		c.hub.difundir <- Mensaje{De: c.nombre, Texto: strings.TrimSpace(string(datos)), Hora: time.Now()}
	}
}

// escribir: la ÚNICA goroutine que escribe en la conexión
func (c *Cliente) escribir() {
	ticker := time.NewTicker(intervaloPing)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case m, ok := <-c.enviar:
			c.conn.SetWriteDeadline(time.Now().Add(tiempoEscritura))
			if !ok { // el hub cerró el channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(m); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(tiempoEscritura))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin protege contra sitios ajenos que abran conexiones con las
	// cookies del usuario. Por defecto exige mismo origen; aquí lo dejamos
	// explícito aceptando localhost.
	CheckOrigin: func(r *http.Request) bool {
		o := r.Header.Get("Origin")
		return o == "" || strings.HasPrefix(o, "http://localhost:") || strings.HasPrefix(o, "http://127.0.0.1:")
	},
}

func (h *Hub) manejarWS(w http.ResponseWriter, r *http.Request) {
	nombre := r.URL.Query().Get("nombre")
	if nombre == "" {
		nombre = "anónimo"
	}
	// Upgrade convierte la conexión HTTP en WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade ya respondió con el error HTTP
	}
	c := &Cliente{hub: h, conn: conn, nombre: nombre, enviar: make(chan Mensaje, 16)}
	h.registrar <- c
	go c.escribir()
	go c.leer()
	h.difundir <- Mensaje{De: "sistema", Texto: nombre + " se unió", Hora: time.Now()}
}

// ======================= SIMULACIÓN =======================

// simularCliente se conecta como lo haría un navegador, usando el Dialer.
// "salida" es un mutex para que las líneas de distintos clientes no se
// mezclen al imprimir.
func simularCliente(url, nombre string, enviar []string, salida *sync.Mutex) {
	conn, resp, err := websocket.DefaultDialer.Dial(url+"?nombre="+nombre, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	salida.Lock()
	fmt.Printf("  %s conectado (status HTTP %d %s)\n", nombre, resp.StatusCode, http.StatusText(resp.StatusCode))
	salida.Unlock()

	// Leer en segundo plano todo lo que llegue
	recibidos := make(chan Mensaje, 32)
	go func() {
		defer close(recibidos)
		for {
			var m Mensaje
			if err := conn.ReadJSON(&m); err != nil {
				return
			}
			recibidos <- m
		}
	}()

	for _, texto := range enviar {
		time.Sleep(50 * time.Millisecond)
		conn.WriteMessage(websocket.TextMessage, []byte(texto))
	}
	time.Sleep(300 * time.Millisecond)

	// Cierre ordenado: enviar un frame de cierre
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "chau"))
	time.Sleep(50 * time.Millisecond)

	var lineas []string
	for len(recibidos) > 0 {
		m := <-recibidos
		lineas = append(lineas, fmt.Sprintf("[%s] %s", m.De, m.Texto))
	}
	salida.Lock()
	fmt.Printf("  %s recibió %d mensajes: %s\n", nombre, len(lineas), strings.Join(lineas, " | "))
	salida.Unlock()
}

const paginaHTML = `<!doctype html>
<meta charset="utf-8"><title>Chat Go</title>
<body style="font-family:sans-serif;max-width:600px;margin:2em auto">
<h2>Chat con WebSockets</h2>
<div id="log" style="border:1px solid #ccc;height:300px;overflow:auto;padding:.5em"></div>
<form id="f"><input id="t" autocomplete="off" style="width:80%"> <button>Enviar</button></form>
<script>
const nombre = prompt("Tu nombre") || "anónimo";
const ws = new WebSocket("ws://" + location.host + "/ws?nombre=" + encodeURIComponent(nombre));
const log = document.getElementById("log");
ws.onmessage = e => {
  const m = JSON.parse(e.data);
  const p = document.createElement("div");
  p.textContent = new Date(m.hora).toLocaleTimeString() + " " + m.de + ": " + m.texto;
  log.appendChild(p); log.scrollTop = log.scrollHeight;
};
document.getElementById("f").onsubmit = e => {
  e.preventDefault(); const t = document.getElementById("t");
  if (t.value) { ws.send(t.value); t.value = ""; }
};
</script>`

func main() {
	hub := nuevoHub()
	go hub.run()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws", hub.manejarWS)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, paginaHTML)
	})

	// ReadHeaderTimeout protege contra clientes que envían los headers muy
	// lento para agotar conexiones (ataque "Slowloris"). No se usan
	// ReadTimeout/WriteTimeout porque cortarían las conexiones WebSocket,
	// que duran mucho; para ellas están los plazos por mensaje de arriba.
	srv := &http.Server{Addr: ":8081", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	time.Sleep(100 * time.Millisecond) // dar tiempo a que el servidor arranque

	// Nota: json.Marshal de Mensaje es lo que viaja por el socket
	ejemplo, _ := json.Marshal(Mensaje{De: "ana", Texto: "hola", Hora: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)})
	fmt.Println("Formato de cada mensaje:", string(ejemplo))

	fmt.Println("\nSimulando 3 clientes conectados al mismo tiempo:")
	var wg sync.WaitGroup
	var salida sync.Mutex
	url := "ws://localhost:8081/ws"
	// Las pausas de 20ms hacen que se conecten en un orden predecible
	wg.Go(func() { simularCliente(url, "ana", []string{"hola a todos"}, &salida) })
	time.Sleep(20 * time.Millisecond)
	wg.Go(func() { simularCliente(url, "bruno", []string{"¿qué tal?"}, &salida) })
	time.Sleep(20 * time.Millisecond)
	wg.Go(func() { simularCliente(url, "carla", nil, &salida) })
	wg.Wait()

	fmt.Println("\nServidor en http://localhost:8081 — abre varias pestañas para chatear (Ctrl+C para salir)")
	select {} // bloquear para siempre
}
