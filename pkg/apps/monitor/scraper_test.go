package monitor

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, true, response.result)
	assert.Equal(t, 1, len(session.scraper.topics))

	_, ok := session.scraper.topics[message.name]
	assert.Equal(t, true, ok)
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

	assert.Equal(t, false, response.result)

	msg := &ScraperUnsubscribeMessage{
		name:     topicName,
		session:  session,
		response: make(chan *ScraperResponseMessage),
	}
	session.scraper.unsubscribe <- msg
	res := <-msg.response

	assert.Equal(t, true, res.result)
	assert.Equal(t, 0, len(session.scraper.topics))
}

func TestBroadcastMessage(t *testing.T) {
	t.Skip("need mock websocket connection")
}
