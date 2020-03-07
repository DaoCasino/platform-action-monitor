package main

import (
	"fmt"
	"log"
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
	// Registered clients.
	sessions map[*Session]bool
	topics   map[string]map[*Session]bool

	register   chan *Session
	unregister chan *Session

	subscribe   chan *ScraperSubscribeMessage
	unsubscribe chan *ScraperUnsubscribeMessage
}

func newScraper() *Scraper {
	return &Scraper{
		sessions:    make(map[*Session]bool),
		topics:      make(map[string]map[*Session]bool),
		register:    make(chan *Session),
		unregister:  make(chan *Session),
		subscribe:   make(chan *ScraperSubscribeMessage),
		unsubscribe: make(chan *ScraperUnsubscribeMessage),
	}
}

func (s *Scraper) run(done <-chan struct{}) {
	log.Print("scraper started")
	for {
		select {
		case <-done:
			log.Print("scraper stopped")
			// TODO: нужно ли очищать что-то ???
			return
		case session := <-s.register:
			s.sessions[session] = true
		case session := <-s.unregister:
			if _, ok := s.sessions[session]; ok {

				for name, topicSessions := range s.topics {
					if _, ok := topicSessions[session]; ok {
						delete(topicSessions, session)
					}

					if len(topicSessions) == 0 {
						delete(s.topics, name)
					}
				}

				delete(s.sessions, session)
				close(session.send)
			}

		case message := <-s.subscribe:
			log.Printf("subscribe message %+v", message)

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
			log.Printf("unsubscribe message %+v", message)

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
