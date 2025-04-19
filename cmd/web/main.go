package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/JamesTiberiusKirk/lambdaban/internal/api/healthcheck"
	"github.com/JamesTiberiusKirk/lambdaban/internal/db"
	"github.com/JamesTiberiusKirk/lambdaban/internal/middleware"
	"github.com/JamesTiberiusKirk/lambdaban/internal/web/index"
	"github.com/JamesTiberiusKirk/lambdaban/internal/web/notifications"
	"github.com/JamesTiberiusKirk/lambdaban/internal/web/todos"
	"github.com/alexedwards/scs/v2"
	"github.com/rickb777/servefiles/v3"
)

func main() {
	logger := slog.Default()

	// Initialize the session.
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour

	db := db.NewInMemClient()

	serverMux := http.NewServeMux()

	nh := notifications.NewNotificationsHandler(logger, sessionManager)
	serverMux.HandleFunc("/notifications", nh.ServeSSE)

	serverMux.Handle("/{$}", index.NewHandler(sessionManager))

	assets := servefiles.NewAssetHandler("./assets/").WithMaxAge(time.Hour)
	serverMux.Handle("/assets/", http.StripPrefix("/assets/", assets))

	serverMux.Handle("/todos", todos.NewHandler(logger, db, sessionManager, nh))
	serverMux.Handle("/api/healthcheck", healthcheck.NewHandler())

	loggedServer := middleware.Logger(logger, serverMux)

	sessionedServer := sessionManager.LoadAndSave(loggedServer)

	logger.Info("HTTP server listening", "port", "3000")

	if err := http.ListenAndServe(":3000", sessionedServer); err != nil {
		logger.Error("failed to start server: ", "error", err)
		return
	}
}
