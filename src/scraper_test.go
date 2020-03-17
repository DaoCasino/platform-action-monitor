package main

import (
	"testing"
)

func TestScraperSubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

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

	if len(session.scraper.topics) != 1 {
		t.Errorf("topics len %d; want 1", len(session.scraper.topics))
	}

	if _, ok := session.scraper.topics[message.name]; !ok {
		t.Error("topic not exists; want true")
	}
}

func TestScraperUnsubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	const topicName = "test"

	session.scraper.subscribe <- &ScraperSubscribeMessage{name: topicName, session: session, response: nil}

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

	msg := &ScraperUnsubscribeMessage{
		name:     topicName,
		session:  session,
		response: make(chan *ScraperResponseMessage),
	}
	session.scraper.unsubscribe <- msg
	res := <-msg.response

	if res.result == false {
		t.Error("unsubscribe result false; want true")
	}

	if len(session.scraper.topics) != 0 {
		t.Errorf("topics len %d; want 0", len(session.scraper.topics))
	}
}

func TestBroadcastMessage(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	const topicName = "test"

	msg := &ScraperBroadcastMessage{
		message:  []byte("test"),
		name:     topicName,
		response: make(chan *ScraperResponseMessage),
	}
	session.scraper.broadcast <- msg
	res := <-msg.response

	if res.result == true {
		t.Error("broadcast result true; want false")
	}

	session.scraper.subscribe <- &ScraperSubscribeMessage{name: topicName, session: session, response: nil}
	msg.response = make(chan *ScraperResponseMessage)

	session.scraper.broadcast <- msg
	res = <-msg.response

	if res.result == false {
		t.Error("broadcast result false; want true")
	}
}
