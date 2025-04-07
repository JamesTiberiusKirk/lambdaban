package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/JamesTiberiusKirk/todoist/internal/api/healthcheck"
	"github.com/JamesTiberiusKirk/todoist/internal/middleware"
	"github.com/JamesTiberiusKirk/todoist/internal/web/db"
	"github.com/JamesTiberiusKirk/todoist/internal/web/index"
	"github.com/JamesTiberiusKirk/todoist/internal/web/notfound"
	"github.com/JamesTiberiusKirk/todoist/internal/web/todos"
	"github.com/a-h/templ"
	"github.com/alexedwards/scs/v2"
)

func main() {
	logger := slog.Default()

	// Initialize the session.
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour

	db := db.NewInMemClient()

	serverMux := http.NewServeMux()

	serverMux.Handle("/", index.NewHandler(sessionManager))
	serverMux.Handle("/todos", todos.NewHandler(logger, db, sessionManager))

	serverMux.Handle("/404", templ.Handler(notfound.PageNotFound(), templ.WithStatus(http.StatusNotFound)))
	serverMux.Handle("/api/healthcheck", healthcheck.NewHandler())

	loggedServer := middleware.Logger(logger, serverMux)

	sessionedServer := sessionManager.LoadAndSave(loggedServer)

	logger.Info("HTTP server listening", "port", "3000")

	if err := http.ListenAndServe(":3000", sessionedServer); err != nil {
		logger.Error("failed to start server: ", "error", err)
		return
	}
}
