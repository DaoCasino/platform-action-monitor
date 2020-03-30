package main

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"net/http"
)

var upgrader websocket.Upgrader

type SessionManager struct {
	sessions   map[*Session]bool
	register   chan *Session
	unregister chan *Session
}

func newSessionManager() *SessionManager {
	upgrader = websocket.Upgrader{
		ReadBufferSize:  config.upgrader.readBufferSize,
		WriteBufferSize: config.upgrader.writeBufferSize,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	return &SessionManager{
		sessions:   make(map[*Session]bool),
		register:   make(chan *Session),
		unregister: make(chan *Session),
	}
}

func (s *SessionManager) run(done <-chan struct{}) {
	defer func() {
		for session := range s.sessions {
			scraper.unsubscribeSession <- session

			delete(s.sessions, session)
			close(session.send)
			usersOnline.Dec()
		}
		sessionLog.Info("session manager stopped")
	}()

	sessionLog.Info("session manager started")

	for {
		select {
		case <-done:
			return
		case session := <-s.register:
			s.sessions[session] = true
			usersOnline.Inc()
		case session := <-s.unregister:
			if _, ok := s.sessions[session]; ok {
				scraper.unsubscribeSession <- session

				delete(s.sessions, session)
				close(session.send)
				usersOnline.Dec()
			}
		}
	}
}

func serveWs(scraper *Scraper, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sessionLog.Error("upgrade", zap.Error(err))
		return
	}

	session := newSession(scraper, conn)
	sessionManager.register <- session

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go session.writePump()
	go session.readPump()
}
