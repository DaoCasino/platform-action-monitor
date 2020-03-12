package main

import (
	"context"
	"flag"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// context DeadLine timeout
const withTimeout = 5 * time.Second

func main() {
	config := newConfig()

	// TODO: delete!
	logger := newLogger(false)
	EnableDebugLogging(logger)
	// -----------

	flag.Parse()

	scraper := newScraper()
	manager := newSessionManager(&config.upgrader)

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(config, scraper, manager, w, r)
	})

	srv := &http.Server{
		Addr:    config.serverAddress,
		Handler: router,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	idleConnectionClosed := make(chan struct{})
	go manager.run(idleConnectionClosed)
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
		//TODO: Close database, redis, truncate message queues, etc.
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		mainLog.Fatal("server shutdown failed", zap.Error(err))
	}
}
