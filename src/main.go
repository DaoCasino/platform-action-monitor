package main

import (
	"context"
	"flag"
	"github.com/jackc/pgx/v4"
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
	registry := newRegistry()
	config := newConfig()

	registry.set(serviceConfig, config)

	// TODO: delete!
	logger := newLogger(false)
	EnableDebugLogging(logger)
	// -----------

	flag.Parse()

	abiDecoder, err := newAbiDecoder(&config.abi)
	if err != nil {
		mainLog.Fatal("abi decoder error", zap.Error(err))
	}
	registry.set(serviceAbiDecoder, abiDecoder)

	db, err := pgx.Connect(context.Background(), config.db.url)
	if err != nil {
		mainLog.Fatal("database connection", zap.Error(err))
	}
	registry.set(serviceDatabase, db)

	fetchEvent := newFetchEvent(registry)
	registry.set(serviceFetchEvent, fetchEvent)

	scraper := newScraper(registry)
	registry.set(serviceScraper, scraper)

	manager := newSessionManager(registry)
	registry.set(serviceSessionManager, manager)

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(registry, w, r)
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
	go scraper.listen(db, &config.db.filter, idleConnectionClosed)

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
		db.Close(context.Background())
		registry.clean()

		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		mainLog.Fatal("server shutdown failed", zap.Error(err))
	}
}
