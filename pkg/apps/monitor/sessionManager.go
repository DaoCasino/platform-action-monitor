package monitor

import (
	"context"
	"github.com/DaoCasino/platform-action-monitor/pkg/apps/monitor/metrics"
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

func (s *SessionManager) run(parentContext context.Context) {
	defer func() {
		for session := range s.sessions {
			scraper.unsubscribeSession <- session

			delete(s.sessions, session)
			close(session.queue)
			metrics.UsersOnline.Dec()
		}
		sessionLog.Info("session manager stopped")
	}()

	sessionLog.Info("session manager started")

	for {
		select {
		case <-parentContext.Done():
			sessionLog.Debug("session manager parent context done")
			return
		case session := <-s.register:
			s.sessions[session] = true
			metrics.UsersOnline.Inc()
		case session := <-s.unregister:
			if _, ok := s.sessions[session]; ok {
				session.scraper.unsubscribeSession <- session

				delete(s.sessions, session)
				close(session.queue)
				metrics.UsersOnline.Dec()
			}
		}
	}
}

func serveWs(parentContext context.Context, scraper *Scraper, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sessionLog.Error("upgrade", zap.Error(err))
		return
	}

	session := newSession(scraper, conn)
	sessionManager.register <- session

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go session.writePump(parentContext)
	go session.readPump(parentContext)
}
