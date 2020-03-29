package main

import (
	"action-monitor/pkg/tools/rungroup"
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	"action-monitor/pkg/apps/monitor"

	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// context DeadLine timeout
const withTimeout = 5 * time.Second

// Globals
var config *monitor.Config
var abiDecoder *monitor.AbiDecoder
var pool *pgxpool.Pool
var scraper *monitor.Scraper
var sessionManager *monitor.SessionManager

func main() {
	configFile := flag.String("config", "", "config file")
	flag.Parse()

	// TODO: need config log level
	logger := monitor.NewLogger(false)
	monitor.EnableDebugLogging(logger)
	// -----------

	var err error
	config = monitor.NewConfig()
	if *configFile != "" {
		err = config.LoadFromFile(configFile)
		if err != nil {
			monitor.MainLog.Fatal("config file error", zap.Error(err))
		}
	} else {
		monitor.MainLog.Info("set default config")
	}

	abiDecoder, err = monitor.NewAbiDecoder(&config.Abi)
	if err != nil {
		monitor.MainLog.Fatal("abi decoder error", zap.Error(err))
	}

	pool, err = pgxpool.Connect(context.Background(), config.Db.Url)
	if err != nil {
		monitor.MainLog.Fatal("database connection", zap.Error(err))
	}

	scraper = monitor.NewScraper()
	sessionManager = monitor.NewSessionManager()

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		monitor.ServeWs(scraper, w, r)
	})

	router.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    config.ServerAddress,
		Handler: router,
	}

	parentCtx, cancelFunc := context.WithCancel(context.Background())
	gracefulShutdown := rungroup.NewNamedGroup(parentCtx, "platform-action-monitor-main")

	gracefulShutdown.AddWithContextNamed("sessionmanager", sessionManager.Run)
	gracefulShutdown.AddWithContextNamed("scraper", scraper.Run)
	gracefulShutdown.AddNamed("http",
		func() error {
			return srv.ListenAndServe()
		}, func(err error) {
			monitor.MainLog.Info("server stopped")

			ctx, cancel := context.WithTimeout(context.Background(), withTimeout)
			defer func() {
				pool.Close()
				cancel()
			}()

			if err := srv.Shutdown(ctx); err != nil {
				monitor.MainLog.Fatal("server shutdown failed", zap.Error(err))
			}
		})

	if err := gracefulShutdown.Run(); err == nil {
		monitor.MainLog.Info("Graceful shutdown is clean")
	} else {
		monitor.MainLog.Error(fmt.Sprintf("Graceful shutdown with error: %v", err))
	}
	cancelFunc()
}
