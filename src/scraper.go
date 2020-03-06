package main

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

func (s *Scraper) run() {
	for {
		select {
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
		}
	}
}
