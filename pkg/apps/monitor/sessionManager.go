package monitor

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader websocket.Upgrader

type SessionManager struct {
	sessions   map[*Session]bool
	register   chan *Session
	unregister chan *Session
}

func NewSessionManager() *SessionManager {
	upgrader = websocket.Upgrader{
		ReadBufferSize:  platform_action_monitor.config.upgrader.readBufferSize,
		WriteBufferSize: platform_action_monitor.config.upgrader.writeBufferSize,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	return &SessionManager{
		sessions:   make(map[*Session]bool),
		register:   make(chan *Session),
		unregister: make(chan *Session),
	}
}

func (s *SessionManager) Run(ctx context.Context) error {
	defer func() {
		for session := range s.sessions {
			platform_action_monitor.scraper.unsubscribeSession <- session

			delete(s.sessions, session)
			close(session.send)
		}
		sessionLog.Info("session manager stopped")
	}()

	sessionLog.Info("session manager started")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case session := <-s.register:
			s.sessions[session] = true
		case session := <-s.unregister:
			if _, ok := s.sessions[session]; ok {
				platform_action_monitor.scraper.unsubscribeSession <- session

				delete(s.sessions, session)
				close(session.send)
			}
		}
	}
}

func ServeWs(scraper *Scraper, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sessionLog.Error("upgrade", zap.Error(err))
		return
	}

	session := newSession(scraper, conn)
	platform_action_monitor.sessionManager.register <- session

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go session.writePump()
	go session.readPump()
}
