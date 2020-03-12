package main

import (
	"encoding/json"
	"go.uber.org/zap"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lucsky/cuid"
)

type Session struct {
	ID     string
	offset int

	config  *SessionConfig
	scraper *Scraper
	manager *SessionManager

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func newSession(config *SessionConfig, scraper *Scraper, manager *SessionManager, conn *websocket.Conn) *Session {
	ID := cuid.New()
	sessionLog.Debug("new session", zap.String("ID", ID))

	return &Session{
		ID:      ID,
		config:  config,
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

		sessionLog.Debug("readPump close", zap.String("session.id", s.ID))
	}()

	s.conn.SetReadLimit(s.config.maxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(s.config.pongWait))
	s.conn.SetPongHandler(func(string) error { s.conn.SetReadDeadline(time.Now().Add(s.config.pongWait)); return nil })
	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				sessionLog.Error("readPump", zap.Error(err))
			}
			break
		}

		s.process(message)
	}
}

func (s *Session) writePump() {
	ticker := time.NewTicker(s.config.pingPeriod)
	defer func() {
		ticker.Stop()
		s.conn.Close()

		sessionLog.Debug("writePump close", zap.String("session.id", s.ID))
	}()
	for {
		select {
		case message, ok := <-s.send:
			s.conn.SetWriteDeadline(time.Now().Add(s.config.writeWait))
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
			s.conn.SetWriteDeadline(time.Now().Add(s.config.writeWait))
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

	raw, err := json.Marshal(response)
	if err != nil {
		return err
	}

	session.send <- raw
	return e
}
