package main

import (
	"context"
	"flag"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// context DeadLine timeout
const withTimeout = 5 * time.Second

// Globals
var config *Config
var abiDecoder *AbiDecoder
var pool *pgxpool.Pool
var scraper *Scraper
var sessionManager *SessionManager

func main() {
	configFile := flag.String("config", "", "config file")
	flag.Parse()

	// TODO: need config log level
	logger := newLogger(false)
	EnableDebugLogging(logger)
	// -----------

	var err error
	config = newConfig()
	if *configFile != "" {
		err = config.loadFromFile(configFile)
		if err != nil {
			mainLog.Fatal("config file error", zap.Error(err))
		}
	} else {
		mainLog.Info("set default config")
	}

	abiDecoder, err = newAbiDecoder(&config.abi)
	if err != nil {
		mainLog.Fatal("abi decoder error", zap.Error(err))
	}

	pool, err = pgxpool.Connect(context.Background(), config.db.url)
	if err != nil {
		mainLog.Fatal("database connection", zap.Error(err))
	}

	scraper = newScraper()
	sessionManager = newSessionManager()

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(scraper, w, r)
	})

	router.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    config.serverAddress,
		Handler: router,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	idleConnectionClosed := make(chan struct{})
	go sessionManager.run(idleConnectionClosed)
	go scraper.run(idleConnectionClosed)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mainLog.Fatal("listen", zap.Error(err))
		}
	}()

	mainLog.Info("server is listening", zap.String("addr", config.serverAddress))

	<-done
	close(idleConnectionClosed)

	mainLog.Info("server stopped")

	ctx, cancel := context.WithTimeout(context.Background(), withTimeout)
	defer func() {
		pool.Close()
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		mainLog.Fatal("server shutdown failed", zap.Error(err))
	}
}
