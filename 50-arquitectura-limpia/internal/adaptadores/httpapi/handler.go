// Package httpapi es un adaptador de ENTRADA: traduce HTTP/JSON a llamadas a
// los casos de uso, y los resultados/errores de vuelta a HTTP/JSON.
// No contiene reglas de negocio: si mañana se agrega una CLI o gRPC, se
// escribe otro adaptador que llama al mismo ServicioPedidos.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang-basico/50-arquitectura-limpia/internal/aplicacion"
	"golang-basico/50-arquitectura-limpia/internal/dominio"
)

// DTOs de transporte: el JSON público es independiente de las entidades.
// Así se puede cambiar el dominio sin romper a los clientes de la API.
type itemJSON struct {
	Producto string  `json:"producto"`
	Cantidad int     `json:"cantidad"`
	Precio   float64 `json:"precio"`
}

type crearPedidoJSON struct {
	Cliente string     `json:"cliente"`
	Items   []itemJSON `json:"items"`
}

type pedidoJSON struct {
	ID       string     `json:"id"`
	Cliente  string     `json:"cliente"`
	Estado   string     `json:"estado"`
	Total    float64    `json:"total"`
	Items    []itemJSON `json:"items"`
	CreadoEn string     `json:"creado_en"`
}

func aJSON(p *dominio.Pedido) pedidoJSON {
	out := pedidoJSON{ID: p.ID, Cliente: p.Cliente, Estado: string(p.Estado),
		Total: p.Total(), CreadoEn: p.CreadoEn.Format(time.DateOnly)}
	for _, it := range p.Items {
		out.Items = append(out.Items, itemJSON{it.Producto, it.Cantidad, it.PrecioUnitario})
	}
	return out
}

type Handler struct{ svc *aplicacion.ServicioPedidos }

func NuevoHandler(svc *aplicacion.ServicioPedidos) http.Handler {
	h := &Handler{svc: svc}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /pedidos", h.crear)
	mux.HandleFunc("GET /pedidos", h.listar)
	mux.HandleFunc("GET /pedidos/{id}", h.obtener)
	mux.HandleFunc("POST /pedidos/{id}/pagar", h.pagar)
	mux.HandleFunc("POST /pedidos/{id}/cancelar", h.cancelar)
	return mux
}

func responder(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// responderError mapea errores de DOMINIO a status HTTP. Es el único lugar
// donde se decide que "no encontrado" es 404.
func responderError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, dominio.ErrPedidoNoEncontrado):
		status = http.StatusNotFound
	case errors.Is(err, dominio.ErrDatosInvalidos):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, dominio.ErrTransicionInvalida):
		status = http.StatusConflict
	}
	responder(w, status, map[string]string{"error": err.Error()})
}

func (h *Handler) crear(w http.ResponseWriter, r *http.Request) {
	var in crearPedidoJSON
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		responder(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	cmd := aplicacion.CrearPedidoInput{Cliente: in.Cliente}
	for _, it := range in.Items {
		cmd.Items = append(cmd.Items, dominio.Item{Producto: it.Producto, Cantidad: it.Cantidad, PrecioUnitario: it.Precio})
	}
	p, err := h.svc.CrearPedido(r.Context(), cmd)
	if err != nil {
		responderError(w, err)
		return
	}
	responder(w, http.StatusCreated, aJSON(p))
}

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	ps, err := h.svc.ListarPedidos(r.Context())
	if err != nil {
		responderError(w, err)
		return
	}
	out := make([]pedidoJSON, 0, len(ps))
	for _, p := range ps {
		out = append(out, aJSON(p))
	}
	responder(w, http.StatusOK, out)
}

func (h *Handler) obtener(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.ObtenerPedido(r.Context(), r.PathValue("id"))
	if err != nil {
		responderError(w, err)
		return
	}
	responder(w, http.StatusOK, aJSON(p))
}

func (h *Handler) pagar(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.PagarPedido(r.Context(), r.PathValue("id"))
	if err != nil {
		responderError(w, err)
		return
	}
	responder(w, http.StatusOK, aJSON(p))
}

func (h *Handler) cancelar(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.CancelarPedido(r.Context(), r.PathValue("id"))
	if err != nil {
		responderError(w, err)
		return
	}
	responder(w, http.StatusOK, aJSON(p))
}
