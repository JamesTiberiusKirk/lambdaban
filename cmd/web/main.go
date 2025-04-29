package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/JamesTiberiusKirk/lambdaban/internal/api/healthcheck"
	"github.com/JamesTiberiusKirk/lambdaban/internal/config"
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
	config := config.GetConfig()

	logger := slog.Default()

	promReg := prometheus.NewRegistry()
	m := metrics.NewMetrics(promReg)

	// Initialize the session.
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour

	db, err := db.InitClient(logger, m,
		config.DbUser, config.DbPass, config.DbHost, config.DbName,
		true, time.Now)
	if err != nil {
		panic("error connecting to db " + err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db.InitTTLCleanup(ctx, 10*time.Minute, 2*time.Hour)

	serverMux := http.NewServeMux()

	nh := notifications.NewNotificationsHandler(logger, m, sessionManager)
	serverMux.HandleFunc("/notifications", nh.ServeSSE)

	serverMux.Handle("/{$}", index.NewHandler(sessionManager))

	serverMux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "assets/lambda.ico")
	})

	assets := servefiles.NewAssetHandler("./assets/").WithMaxAge(time.Hour)
	serverMux.Handle("/assets/", http.StripPrefix("/assets/", assets))

	todosHandler := todos.NewHandler(logger, db, sessionManager, nh)
	serverMux.Handle("/todos", todosHandler)
	serverMux.Handle("/todos/", todosHandler)

	serverMux.Handle("/api/healthcheck", healthcheck.NewHandler())

	loggedServer := metrics.HTTPMiddleware(m, middleware.Logger(logger, serverMux))
	// loggedServer := middleware.Logger(logger, serverMux)

	sessionedServer := sessionManager.LoadAndSave(loggedServer)

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "3031"
	}

	// Serve metrics.
	logger.Info("serving metrics at:", "metrics_port", metricsPort)
	go http.ListenAndServe(":"+metricsPort, promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3030"
	}

	logger.Info("HTTP server listening", "port", port)
	if err := http.ListenAndServe(":"+port, sessionedServer); err != nil {
		logger.Error("failed to start server: ", "error", err)
		return
	}
}
