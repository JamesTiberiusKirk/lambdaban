package todos

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/JamesTiberiusKirk/todoist/internal/models"
	"github.com/JamesTiberiusKirk/todoist/internal/web/db"
	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
)

func NewHandler(log *slog.Logger, db *db.InMemClient, sm *scs.SessionManager) http.Handler {
	return &handler{
		log: log,
		db:  db,
		sm:  sm,
	}
}

type handler struct {
	log *slog.Logger
	db  *db.InMemClient
	sm  *scs.SessionManager
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
		return
	case "POST":
		h.post(w, r)
		return
	case "DELETE":
		h.delete(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	userId := h.sm.GetString(r.Context(), "user")
	r.ParseForm()

	todoId := r.Form.Get("todo_id")

	h.log.Info("deleting", "userid", userId, "todoId", todoId)

	err := h.db.DeleteTodoByUserAndTodoId(userId, todoId)
	if err != nil {
		h.log.Error("Error deleting todo ", "userId", userId, "todoId", todoId, "err", err.Error())
	}

	h.get(w, r)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	todos := []models.Todos{}
	userId := h.sm.GetString(r.Context(), "user")
	if userId == "" {
		userId = uuid.NewString()
	} else {
		todos, _ = h.db.GetAllByUser(userId)
	}

	component := page(userId, todos)
	component.Render(r.Context(), w)
}

func (h *handler) post(w http.ResponseWriter, r *http.Request) {
	userId := h.sm.GetString(r.Context(), "user")
	if userId == "" {
		userId = uuid.NewString()
	}

	// Update state.
	r.ParseForm()

	newTodo := models.Todos{}

	newTodo.Title = r.Form.Get("title")
	newTodo.Description = r.Form.Get("description")
	newTodo.Status = models.StatusTodo
	newTodo.CreatedAt = time.Now()
	newTodo.LastUpdatedAt = time.Now()
	newTodo.Id = uuid.NewString()

	err := h.db.AddToUser(userId, newTodo)
	if err != nil {
		// TODO
	}

	h.sm.Put(r.Context(), "user", userId)

	// Display the form.
	h.get(w, r)
}
