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

var addr = flag.String("addr", ":8888", "http service address")

func main() {

	// TODO: delete!
	logger, _ := zap.NewDevelopment()
	EnableDebugLogging(logger)
	// -----------

	flag.Parse()

	scraper := newScraper()
	manager := newSessionManager()

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(scraper, manager, w, r)
	})

	srv := &http.Server{
		Addr:    *addr,
		Handler: router,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	idleConnectionClosed := make(chan struct{})
	go manager.run(idleConnectionClosed)
	go scraper.run(idleConnectionClosed)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if loggingEnabled {
				mainLog.Fatal("listen", zap.Error(err))
			}
			os.Exit(1)
		}
	}()

	if loggingEnabled {
		mainLog.Info("server is listening", zap.Stringp("addr", addr))
	}

	<-done

	close(idleConnectionClosed)

	if loggingEnabled {
		mainLog.Info("server stopped")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		//TODO: Close database, redis, truncate message queues, etc.
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		if loggingEnabled {
			mainLog.Fatal("server shutdown failed", zap.Error(err))
		}
		os.Exit(1)
	}
}
