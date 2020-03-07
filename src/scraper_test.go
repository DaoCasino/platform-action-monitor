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

func TestSubscribe(t *testing.T) {
	scraper, teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	session := &Session{ID: cuid.New(), scraper: scraper, conn: nil, send: make(chan []byte, 512)}
	session.scraper.register <- session

	message := &ScraperSubscribeMessage{
		name:     "test",
		session:  session,
		response: make(chan *ScraperResponseMessage),
	}

	session.scraper.subscribe <- message
	response := <-message.response

	if response.result != true {
		t.Error("subscribe response false; want true")
	}

	if len(scraper.topics) != 1 {
		t.Errorf("topics len %d; want 1", len(scraper.topics))
	}

	if _, ok := scraper.topics[message.name]; !ok {
		t.Error("topic not exists; want true")
	}
}

func TestUnsubscribe(t *testing.T) {
	scraper, teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	session := &Session{ID: cuid.New(), scraper: scraper, conn: nil, send: make(chan []byte, 512)}

	session.scraper.register <- session
	session.scraper.subscribe <- &ScraperSubscribeMessage{name: "test", session: session, response: nil}

	message := &ScraperUnsubscribeMessage{
		name:     "123",
		session:  session,
		response: make(chan *ScraperResponseMessage),
	}

	session.scraper.unsubscribe <- message
	response := <-message.response

	if response.result == true {
		t.Error("unsubscribe not exists topic result true; want false")
	}

	message.name = "test"
	message.response = make(chan *ScraperResponseMessage)
	session.scraper.unsubscribe <- message
	res := <-message.response

	if res.result == false {
		t.Error("unsubscribe result false; want true")
	}

	if len(scraper.topics) != 0 {
		t.Errorf("topics len %d; want 0", len(scraper.topics))
	}
}
