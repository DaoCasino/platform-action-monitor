package main

import (
	"github.com/lucsky/cuid"
	"testing"
	"time"
)

func setupTestCase(t *testing.T) (*Scraper, func(t *testing.T)) {
	scraper := newScraper()
	done := make(chan struct{})
	go scraper.run(done)

	t.Log("scraper running")

	return scraper, func(t *testing.T) {
		t.Log("scraper stopped")
		close(done)
	}
}

func TestRegisterUnregister(t *testing.T) {
	scraper, teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	session := &Session{ID: cuid.New(), scraper: scraper, conn: nil, send: make(chan []byte, 512)}
	session.scraper.register <- session

	if _, ok := scraper.sessions[session]; !ok {
		t.Error("register session not exists; want true")
	}

	session.scraper.unregister <- session
	time.Sleep(20 * time.Microsecond) // TODO: дебелизм, не понятно через сколько сессия удалиться
	if _, ok := scraper.sessions[session]; ok {
		t.Error("unregister session exists; want false")
	}
}
