package main

import (
	"log/slog"
	"net/http"
	"os"
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
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	logger := slog.Default()

	// Initialize the session.
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour

	db := db.NewInMemClient(logger)

	serverMux := http.NewServeMux()

	nh := notifications.NewNotificationsHandler(logger, sessionManager)
	serverMux.HandleFunc("/notifications", nh.ServeSSE)

	serverMux.Handle("/{$}", index.NewHandler(sessionManager))

	serverMux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "assets/lambda.ico")
	})

	assets := servefiles.NewAssetHandler("./assets/").WithMaxAge(time.Hour)
	serverMux.Handle("/assets/", http.StripPrefix("/assets/", assets))

	serverMux.Handle("/todos", todos.NewHandler(logger, db, sessionManager, nh))
	serverMux.Handle("/api/healthcheck", healthcheck.NewHandler())

	loggedServer := middleware.Logger(logger, serverMux)

	sessionedServer := sessionManager.LoadAndSave(loggedServer)

	logger.Info("HTTP server listening", "port", port)

	if err := http.ListenAndServe(":"+port, sessionedServer); err != nil {
		logger.Error("failed to start server: ", "error", err)
		return
	}
}
