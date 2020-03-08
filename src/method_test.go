package main

import (
	"github.com/lucsky/cuid"
	"testing"
)


func setupMethodTestCase(t *testing.T) (*Session, func(t *testing.T)) {
	scraper := newScraper()
	done := make(chan struct{})
	go scraper.run(done)
	t.Log("scraper running")

	session := &Session{ID: cuid.New(), scraper: scraper, conn: nil, send: make(chan []byte, 512)}
	session.scraper.register <- session
	t.Log("session register")

	return session, func(t *testing.T) {
		session.scraper.unregister <- session
		t.Log("session unregister")
		close(done)
		t.Log("scraper stopped")
	}
}

func TestMethodSubscribe(t *testing.T) {
	session, teardownTestCase := setupMethodTestCase(t)
	defer teardownTestCase(t)

	subscribe := &methodSubscribeParams{Topic: "test"}
	result, err := subscribe.execute(session)

	if err == nil && result == false {
		t.Error("subscribe method result false; want true")
	}

	if err != nil {
		t.Error(err)
	}
}

func TestMethodUnsubscribe(t *testing.T) {
	session, teardownTestCase := setupMethodTestCase(t)
	defer teardownTestCase(t)

	const topicName = "test"

	unsubscribe := &methodUnsubscribeParams{Topic: topicName}
	result, err := unsubscribe.execute(session)

	if err == nil || result == true {
		t.Error("unsubscribe method error nil or result true; want error message and false")
	}

	subscribe := &methodSubscribeParams{Topic: topicName}
	subscribe.execute(session)

	result, err = unsubscribe.execute(session)

	if err == nil && result == false {
		t.Error("unsubscribe method result false; want true")
	}

	if err != nil {
		t.Error(err)
	}
}