package monitor

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
	"net/http"
	"github.com/DaoCasino/platform-action-monitor/pkg/apps/monitor/metrics"
)

// Globals
var (
	config         *Config
	abiDecoder     *AbiDecoder
	pool           *pgxpool.Pool
	scraper        *Scraper
	sessionManager *SessionManager
)

func Init(configFile *string, parentContext context.Context) (*http.Server, func(),  error) {
	logger := newLogger(false)
	EnableDebugLogging(logger)

	var err error
	config = newConfig()
	if *configFile != "" {
		err = config.loadFromFile(configFile)
		if err != nil {
			return nil, nil, fmt.Errorf("config file error: %s", err.Error())
		}
	}

	abiDecoder, err = newAbiDecoder(&config.abi)
	if err != nil {
		return nil, nil, fmt.Errorf("abi decoder error: %s", err.Error())
	}

	pool, err = pgxpool.Connect(parentContext, config.db.url)
	if err != nil {
		return nil, nil, fmt.Errorf("database connection error: %s", err.Error())
	}

	scraper = newScraper()
	sessionManager = newSessionManager()

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWs(parentContext, scraper, w, r)
	})

	metrics.Handle(router)

	srv := &http.Server{
		Addr:    config.serverAddress,
		Handler: router,
	}

	//done := make(chan os.Signal, 1)
	//signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go sessionManager.run(parentContext)
	go scraper.run(parentContext)

	//go func() {
	//	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	//		mainLog.Fatal("listen", zap.Error(err))
	//	}
	//}()

	//mainLog.Info("server is listening", zap.String("addr", config.serverAddress))

	//<-done
	//mainLog.Debug("done signal")
	//mainCancel()

	//shutdownContextWithTimeout, cancelWaitShutdown := context.WithTimeout(parentContext, withTimeout)
	//defer func() {
	//	// TODO: close all connections here
	//	pool.Close()
	//
	//	cancelWaitShutdown()
	//	mainLog.Debug("connection closed")
	//}()

	//if err := srv.Shutdown(shutdownContextWithTimeout); err != nil {
	//	mainLog.Fatal("server shutdown failed", zap.Error(err))
	//}

	closeFunc := func() {
		pool.Close()
	}

	return srv, closeFunc, nil
}