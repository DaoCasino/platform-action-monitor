package main

import (
	"context"
	"flag"
	"github.com/DaoCasino/platform-action-monitor/pkg/apps/monitor"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const withTimeout = 5 * time.Second

func main() {
	configFile := flag.String("config", "", "config file")
	flag.Parse()

	parentContext := context.Background()
	mainContext, mainCancel := context.WithCancel(parentContext)

	server, monitorCancelFunc, err := monitor.Init(configFile, mainContext)
	if err != nil {
		log.Fatalf("monitor init: %s", err.Error())
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s", err.Error())
		}
	}()

	log.Printf("server is listening on %s", server.Addr)
	<-done
	log.Println("done signal")
	mainCancel()

	shutdownContextWithTimeout, cancelWaitShutdown := context.WithTimeout(parentContext, withTimeout)

	defer func() {
		// TODO: close all connections here
		monitorCancelFunc()

		cancelWaitShutdown()
		log.Println("connection closed")
	}()

	if err := server.Shutdown(shutdownContextWithTimeout); err != nil {
		log.Fatalf("server shutdown failed: %s", err.Error())
	}
}
