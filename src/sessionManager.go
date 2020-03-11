package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type SessionManager struct {
	sessions   map[*Session]bool
	register   chan *Session
	unregister chan *Session
}

func newSessionManager() *SessionManager {
	return &SessionManager{
		sessions:   make(map[*Session]bool),
		register:   make(chan *Session),
		unregister: make(chan *Session),
	}
}

func (s *SessionManager) run(done <-chan struct{}) {
	defer func() {
		for session := range s.sessions {
			session.scraper.unsubscribeSession <- session

			delete(s.sessions, session)
			close(session.send)
		}
	}()

	log.Print("session manager started")
	for {
		select {
		case <-done:
			log.Print("session manager stopped")
			return
		case session := <-s.register:
			s.sessions[session] = true
		case session := <-s.unregister:
			if _, ok := s.sessions[session]; ok {
				session.scraper.unsubscribeSession <- session

				delete(s.sessions, session)
				close(session.send)
			}
		}
	}
}

func serveWs(scraper *Scraper, manager *SessionManager, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	session := newSession(scraper, manager, conn)
	manager.register <- session

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go session.writePump()
	go session.readPump()
}
