package main

import (
	"testing"
)

func setupTestCase(t *testing.T) (*Scraper, func(t *testing.T)) {
	scraper := newScraper()
	done := make(chan struct{})
	go scraper.run(done)

	t.Log("scraper running")

	return scraper, func(t *testing.T) {
		t.Log("srcapper stopped")
		close(done)
	}
}

func TestRegister(t *testing.T) {
	_, teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	// session := &Session{ID: cuid.New(), scraper: scraper, conn: nil, send: nil}

}
