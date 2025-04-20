package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/JamesTiberiusKirk/lambdaban/internal/api/healthcheck"
	"github.com/JamesTiberiusKirk/lambdaban/internal/db"
	"github.com/JamesTiberiusKirk/lambdaban/internal/metrics"
	"github.com/JamesTiberiusKirk/lambdaban/internal/middleware"
	"github.com/JamesTiberiusKirk/lambdaban/internal/web/index"
	"github.com/JamesTiberiusKirk/lambdaban/internal/web/notifications"
	"github.com/JamesTiberiusKirk/lambdaban/internal/web/todos"
	"github.com/alexedwards/scs/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rickb777/servefiles/v3"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3030"
	}

	logger := slog.Default()

	promReg := prometheus.NewRegistry()
	m := metrics.NewMetrics(promReg)

	// Initialize the session.
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour

	db := db.NewInMemClient(logger, m)
	db.InitTTLCleanup()

	serverMux := http.NewServeMux()

	serverMux.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))

	nh := notifications.NewNotificationsHandler(logger, m, sessionManager)
	serverMux.HandleFunc("/notifications", nh.ServeSSE)

	serverMux.Handle("/{$}", index.NewHandler(sessionManager))

	serverMux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "assets/lambda.ico")
	})

	assets := servefiles.NewAssetHandler("./assets/").WithMaxAge(time.Hour)
	serverMux.Handle("/assets/", http.StripPrefix("/assets/", assets))

	serverMux.Handle("/todos", todos.NewHandler(logger, db, sessionManager, nh))
	serverMux.Handle("/api/healthcheck", healthcheck.NewHandler())

	loggedServer := metrics.HTTPMiddleware(m, middleware.Logger(logger, serverMux))
	// loggedServer := middleware.Logger(logger, serverMux)

	sessionedServer := sessionManager.LoadAndSave(loggedServer)

	logger.Info("HTTP server listening", "port", port)

	if err := http.ListenAndServe(":"+port, sessionedServer); err != nil {
		logger.Error("failed to start server: ", "error", err)
		return
	}
}
