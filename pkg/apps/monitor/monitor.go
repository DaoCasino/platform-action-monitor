package monitor

import (
	"context"
	"fmt"
	"github.com/DaoCasino/platform-action-monitor/pkg/apps/monitor/metrics"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
	"net/http"
)

// Globals
var (
	config         *Config
	abiDecoder     *AbiDecoder
	pool           *pgxpool.Pool
	scraper        *Scraper
	sessionManager *SessionManager
)

func Init(configFile *string, parentContext context.Context) (*http.Server, func(), error) {
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

	router.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})

	metrics.Handle(router)

	srv := &http.Server{
		Addr:    config.serverAddress,
		Handler: router,
	}

	go sessionManager.run(parentContext)
	go scraper.run(parentContext)

	closeFunc := func() {
		pool.Close()
	}

	return srv, closeFunc, nil
}
