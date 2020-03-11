package main

import (
	"fmt"
	"go.uber.org/zap"
)

type ScraperSubscribeMessage struct {
	name     string
	session  *Session
	response chan *ScraperResponseMessage
}

type ScraperUnsubscribeMessage struct {
	name     string
	session  *Session
	response chan *ScraperResponseMessage
}

type ScraperResponseMessage struct {
	result interface{}
	err    error
}

type Scraper struct {
	topics             map[string]map[*Session]bool
	unsubscribeSession chan *Session
	subscribe          chan *ScraperSubscribeMessage
	unsubscribe        chan *ScraperUnsubscribeMessage
}

func newScraper() *Scraper {
	return &Scraper{
		topics:             make(map[string]map[*Session]bool),
		subscribe:          make(chan *ScraperSubscribeMessage),
		unsubscribe:        make(chan *ScraperUnsubscribeMessage),
		unsubscribeSession: make(chan *Session),
	}
}

func (s *Scraper) run(done <-chan struct{}) {
	defer func() {
		if loggingEnabled {
			scraperLog.Info("scraper stopped")
		}
	}()

	if loggingEnabled {
		scraperLog.Info("scraper started")
	}

	for {
		select {
		case <-done:
			return

		case session := <-s.unsubscribeSession:
			for name, topicSessions := range s.topics {
				if _, ok := topicSessions[session]; ok {
					delete(topicSessions, session)
				}

				if len(topicSessions) == 0 {
					delete(s.topics, name)
				}
			}

		case message := <-s.subscribe:
			if loggingEnabled {
				scraperLog.Debug("subscribe",
					zap.String("name", message.name),
					zap.String("session.id", message.session.ID),
				)
			}

			if topicClients, ok := s.topics[message.name]; ok {
				topicClients[message.session] = true
			} else {
				topicClients := make(map[*Session]bool)
				topicClients[message.session] = true
				s.topics[message.name] = topicClients
			}

			if message.response != nil {
				response := new(ScraperResponseMessage)
				response.result = true
				message.response <- response
				close(message.response)
			}

		case message := <-s.unsubscribe:
			if loggingEnabled {
				scraperLog.Debug("unsubscribe",
					zap.String("name", message.name),
					zap.String("session.id", message.session.ID),
				)
			}

			response := new(ScraperResponseMessage)
			if topicClients, ok := s.topics[message.name]; ok {
				delete(topicClients, message.session)
				if len(topicClients) == 0 {
					delete(s.topics, message.name)
				}
				response.result = true
			} else {
				response.result = false
				response.err = fmt.Errorf("topic %s not exist", message.name)
			}

			if message.response != nil {
				message.response <- response
				close(message.response)
			}
		}
	}
}
