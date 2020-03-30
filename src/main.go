package main

import (
	"context"
	"flag"
	"github.com/DaoCasino/platform-action-monitor/src/metrics"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const withTimeout = 5 * time.Second

// Globals
var (
	config         *Config
	abiDecoder     *AbiDecoder
	pool           *pgxpool.Pool
	scraper        *Scraper
	sessionManager *SessionManager
)

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

	parentContext := context.Background()
	mainContext, mainCancel := context.WithCancel(parentContext)

	pool, err = pgxpool.Connect(mainContext, config.db.url)
	if err != nil {
		mainCancel()
		mainLog.Fatal("database connection", zap.Error(err))
	}

	scraper = newScraper()
	sessionManager = newSessionManager()

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(mainContext, scraper, w, r)
	})

	metrics.Handle(router)

	srv := &http.Server{
		Addr:    config.serverAddress,
		Handler: router,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go sessionManager.run(mainContext)
	go scraper.run(mainContext)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mainLog.Fatal("listen", zap.Error(err))
		}
	}()

	mainLog.Info("server is listening", zap.String("addr", config.serverAddress))

	<-done
	mainLog.Debug("done signal")
	mainCancel()

	shutdownContextWithTimeout, cancelWaitShutdown := context.WithTimeout(parentContext, withTimeout)
	defer func() {
		// TODO: close all connections here
		pool.Close()

		cancelWaitShutdown()
		mainLog.Debug("connection closed")
	}()

	if err := srv.Shutdown(shutdownContextWithTimeout); err != nil {
		mainLog.Fatal("server shutdown failed", zap.Error(err))
	}
}
