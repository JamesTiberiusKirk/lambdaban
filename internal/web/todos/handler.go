package todos

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/JamesTiberiusKirk/lambdaban/internal/components"
	"github.com/JamesTiberiusKirk/lambdaban/internal/models"
	"github.com/JamesTiberiusKirk/lambdaban/internal/web/notifications"
	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
)

type dbClient interface {
	AddToUser(ctx context.Context, id string, ticket models.Ticket) error
	CreateUser(ctx context.Context) (string, error)
	DeleteTodoByUserAndTodoId(ctx context.Context, userId string, todoId string) error
	GetAllByUser(ctx context.Context, id string) ([]models.Ticket, error)
	GetAllByUserSplitByStatus(ctx context.Context, id string) (todo []models.Ticket, inProgress []models.Ticket, done []models.Ticket, err error)
	UpdateUser(ctx context.Context, userId string, tickets []models.Ticket) error
}

func NewHandler(log *slog.Logger, db dbClient, sm *scs.SessionManager, nh *notifications.NotificationsHandler) http.Handler {
	return &handler{
		log: log,
		db:  db,
		sm:  sm,
		nh:  nh,
	}
}

type handler struct {
	version string
	log     *slog.Logger
	db      dbClient
	sm      *scs.SessionManager
	nh      *notifications.NotificationsHandler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.String() == "/todos/" {
		http.Redirect(w, r, "/todos", http.StatusPermanentRedirect)
		return
	}

	if r.Method == http.MethodGet && r.URL.String() == "/todos/session-reset" {
		h.sessionReset(w, r)
		return
	}

	if r.URL.String() != "/todos" && !strings.HasPrefix(r.URL.String(), "/todos?") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		h.get(w, r)
		return
	case "POST":
		h.post(w, r)
		return
	case "PUT":
		h.put(w, r)
		return
	case "DELETE":
		h.delete(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (h *handler) sessionReset(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Resetting session")

	h.sm.Remove(r.Context(), "user")
	h.get(w, r)
	return
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	userId := h.sm.GetString(r.Context(), "user")
	r.ParseForm()

	todoId := r.Form.Get("todo_id")

	h.log.Info("deleting", "userid", userId, "todoId", todoId)

	err := h.db.DeleteTodoByUserAndTodoId(r.Context(), userId, todoId)
	if err != nil {
		h.log.Error("Error deleting todo ", "userId", userId, "todoId", todoId, "err", err.Error())
	}

	h.nh.Notify(userId, notifications.Notification{
		Type:    "Info",
		Content: fmt.Sprintf("Removed ticket %s", todoId),
	})

	h.get(w, r)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	userId := h.sm.GetString(r.Context(), "user")
	if userId == "" {
		h.log.Info("Creating new user")
		uid, err := h.db.CreateUser(r.Context())
		if err != nil {
			h.log.Error("Unable to create user", "error", err)
			component := components.ServerError(r, "Unable to create user")
			component.Render(r.Context(), w)
			return
		}

		userId = uid

		h.sm.Put(r.Context(), "user", userId)
		h.nh.Notify(userId, notifications.Notification{
			Type:    "Info",
			Content: "New user",
		})
	}

	todo, inProgrss, done, err := h.db.GetAllByUserSplitByStatus(r.Context(), userId)
	if err != nil {
		h.nh.Notify(userId, notifications.Notification{
			Type:    "Error",
			Content: "Error fetching tickets",
		})
	}

	h.log.Info("todos", "len", len(todo), "userid", userId)

	w.WriteHeader(http.StatusOK)
	component := page(r, userId, todo, inProgrss, done)
	component.Render(r.Context(), w)
}

func (h *handler) post(w http.ResponseWriter, r *http.Request) {
	defer h.get(w, r)

	userId := h.sm.GetString(r.Context(), "user")
	if userId == "" {
		h.nh.Notify(userId, notifications.Notification{
			Type:    "Error",
			Content: "Error getting uid from session",
		})
		return
	}

	// Update state.
	err := r.ParseForm()
	if err != nil {
		h.log.Error("Error parsing form", "error", err.Error())
		h.nh.Notify(userId, notifications.Notification{
			Type:    "Error",
			Content: "Error parsing form",
		})
		return
	}

	newTodo := models.Ticket{}

	newTodo.Title = r.Form.Get("title")
	newTodo.Description = r.Form.Get("description")
	newTodo.Status = models.Status(r.Form.Get("status"))
	newTodo.CreatedAt = time.Now()
	newTodo.LastUpdatedAt = time.Now()
	newTodo.Id = uuid.NewString()

	err = h.db.AddToUser(r.Context(), userId, newTodo)
	if err != nil {
		h.log.Error("Error adding tickets", "error", err.Error())
		h.nh.Notify(userId, notifications.Notification{
			Type:    "Error",
			Content: "Error adding ticket",
		})
		return
	}

	h.sm.Put(r.Context(), "user", userId)

	h.nh.Notify(userId, notifications.Notification{
		Type:    "Info",
		Content: fmt.Sprintf("Added ticket %s", newTodo.Id),
	})
}

func (h *handler) put(w http.ResponseWriter, r *http.Request) {
	defer h.get(w, r)

	userId := h.sm.GetString(r.Context(), "user")
	if userId == "" {
		h.nh.Notify(userId, notifications.Notification{
			Type:    "Error",
			Content: "Error userId from session",
		})
		return
	}

	err := r.ParseForm()
	if err != nil {
		h.log.Error("Error parsing form", "error", err.Error())
		h.nh.Notify(userId, notifications.Notification{
			Type:    "Error",
			Content: "Error parsing form",
		})
		return
	}

	ids := r.Form["id"]
	statuss := r.Form["status"]

	if len(ids) != len(statuss) {
		h.log.Error("error id arr is not the same len as status")
		return
	}

	currentTickets, err := h.db.GetAllByUser(r.Context(), userId)
	if err != nil {
		h.log.Error("error getting all user tickets", "error", err.Error())
		h.nh.Notify(userId, notifications.Notification{
			Type:    "Error",
			Content: "Error fetching tickets",
		})
		return
	}

	updatedTickets := []models.Ticket{}

	// to avoid dups
	done := map[string]bool{}
	for i, id := range ids {
		for _, t := range currentTickets {
			if t.Id != id {
				continue
			}

			if done[t.Id] {
				continue
			}

			t.Status = models.Status(statuss[i])
			done[t.Id] = true

			updatedTickets = append(updatedTickets, t)
		}
	}

	err = h.db.UpdateUser(r.Context(), userId, updatedTickets)
	if err != nil {
		h.log.Error("error updating tickets", "error", err.Error())
		h.nh.Notify(userId, notifications.Notification{
			Type:    "Error",
			Content: "Error updating tickets",
		})
		return
	}
}
