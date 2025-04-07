package index

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
)

type GlobalState struct {
	Count int
}

var global GlobalState

func NewHandler(sm *scs.SessionManager) http.Handler {
	return &handler{
		sm: sm,
	}
}

type handler struct {
	sm *scs.SessionManager
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
		return
	case "POST":
		h.post(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	userCount := h.sm.GetInt(r.Context(), "count")
	component := page(global.Count, userCount)
	component.Render(r.Context(), w)
}

func (h *handler) post(w http.ResponseWriter, r *http.Request) {
	// Update state.
	r.ParseForm()

	// Check to see if the global button was pressed.
	if r.Form.Has("global") {
		global.Count++
	}
	if r.Form.Has("user") {
		currentCount := h.sm.GetInt(r.Context(), "count")
		h.sm.Put(r.Context(), "count", currentCount+1)
	}

	// Display the form.
	h.get(w, r)
}
