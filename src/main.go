package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

var addr = flag.String("addr", ":8888", "http service address")

func main() {
	flag.Parse()

	scraper := newScraper()

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(scraper, w, r)
	})

	srv := &http.Server{
		Addr:    *addr,
		Handler: router,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	idleConnectionClosed := make(chan struct{})
	go scraper.run(idleConnectionClosed)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Printf("server is listening on %s\n", *addr)

	<-done
	log.Print("server stopped")
	close(idleConnectionClosed)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		//TODO: Close database, redis, truncate message queues, etc.
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed:%+v", err)
	}
}
