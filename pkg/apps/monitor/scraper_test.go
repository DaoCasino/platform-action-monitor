package monitor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScraperSubscribe(t *testing.T) {
	// session, teardownTestCase := setupSessionTestCase(t)
	// defer teardownTestCase(t)

	scraper := newScraper()
	session := newSession(scraper, nil)
	message := &ScraperSubscribeMessage{
		name:     "test",
		session:  session,
		response: make(chan *ScraperResponseMessage),
	}

	parentContext, cancel := context.WithCancel(context.Background())
	go scraper.run(parentContext)

	session.scraper.subscribe <- message
	response := <-message.response

	assert.Equal(t, true, response.result)

	cancel()
	// assert.Equal(t, 1, len(session.scraper.topics))

	// _, ok := session.scraper.topics[message.name]
	// assert.Equal(t, true, ok)
}

func TestScraperUnsubscribe(t *testing.T) {
	const topicName = "test"

	scraper := newScraper()
	session := newSession(scraper, nil)
	subscribeMessage := &ScraperSubscribeMessage{name: topicName, session: session, response: nil}

	parentContext, cancel := context.WithCancel(context.Background())
	go scraper.run(parentContext)

	session.scraper.subscribe <- subscribeMessage

	unsubscribeMessage := &ScraperUnsubscribeMessage{
		name:     "123",
		session:  session,
		response: make(chan *ScraperResponseMessage),
	}

	session.scraper.unsubscribe <- unsubscribeMessage
	response := <-unsubscribeMessage.response

	assert.Equal(t, false, response.result)

	//unsubscribeMessage.name = topicName
	//unsubscribeMessage.response = make(chan *ScraperResponseMessage)
	//
	//session.scraper.unsubscribe <- unsubscribeMessage
	//res := <-unsubscribeMessage.response
	//
	//assert.Equal(t, true, res.result)

	cancel()
	// assert.Equal(t, 0, len(session.scraper.topics))
}

func TestBroadcastMessage(t *testing.T) {
	t.Skip("need mock websocket connection")
}
