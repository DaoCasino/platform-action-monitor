package main

import (
	"encoding/json"
	"go.uber.org/zap"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lucsky/cuid"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 4
)

type Session struct {
	ID      string
	offset  int
	scraper *Scraper
	manager *SessionManager

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func newSession(scraper *Scraper, manager *SessionManager, conn *websocket.Conn) *Session {
	ID := cuid.New()

	if loggingEnabled {
		sessionLog.Debug("new session", zap.String("ID", ID))
	}

	return &Session{
		ID:      ID,
		scraper: scraper,
		manager: manager,
		conn:    conn,
		send:    make(chan []byte, 512)}
}

func (s *Session) setOffset(offset int) {
	s.offset = offset
}

func (s *Session) readPump() {
	defer func() {
		s.manager.unregister <- s
		s.conn.Close()

		if loggingEnabled {
			sessionLog.Debug("readPump close", zap.String("session.id", s.ID))
		}
	}()
	s.conn.SetReadLimit(maxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error { s.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				if loggingEnabled {
					sessionLog.Error("readPump", zap.Error(err))
				}
			}
			break
		}

		s.process(message)
	}
}

func (s *Session) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		s.conn.Close()

		if loggingEnabled {
			sessionLog.Debug("writePump close", zap.String("session.id", s.ID))
		}
	}()
	for {
		select {
		case message, ok := <-s.send:
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The session closed the channel.
				s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := s.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// FIX: I do not know how to do better, there are 2 types of errors
func (session *Session) process(message []byte) (e error) {
	request := new(RequestMessage)
	response := newResponseMessage()

	if err := json.Unmarshal(message, request); err != nil {
		response.parseError()
		e = err
	} else {
		response.ID = request.ID

		method, err := methodExecutorFactory(*request.Method)

		if err != nil {
			response.methodNotFound()
		} else {
			if len(request.Params) == 0 {
				response.invalidParams()
			} else {
				err := json.Unmarshal(request.Params, &method)
				if err != nil {
					response.parseError()
					e = err
				} else {
					if method.isValid() {
						result, err := method.execute(session)
						if err != nil {
							response.setError(err)
						} else {
							err := response.setResult(result)
							if err != nil {
								response.parseError()
								e = err
							}
						}
					} else {
						response.invalidParams()
					}
				}
			}
		}
	}

	if loggingEnabled {
		if response.Error != nil {
			sessionLog.Debug("response error",
				zap.Stringp("ID", response.ID),
				zap.Int("code", response.Error.Code),
				zap.String("message", response.Error.Message),
			)
		} else {
			sessionLog.Debug("response",
				zap.Stringp("ID", response.ID),
				zap.String("result", string(response.Result)),
			)
		}
	}

	raw, err := json.Marshal(response)
	if err != nil {
		return err
	}

	session.send <- raw
	return e
}
