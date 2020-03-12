package main

import (
	"encoding/json"
	"fmt"
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
		if err := s.conn.Close(); err != nil {
			// sessionLog.Error("connection close error", zap.String("session.id", s.ID), zap.Error(err))
		}

		sessionLog.Debug("readPump close", zap.String("session.id", s.ID))
	}()

	s.conn.SetReadLimit(s.config.maxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(s.config.pongWait))
	s.conn.SetPongHandler(func(string) error { s.conn.SetReadDeadline(time.Now().Add(s.config.pongWait)); return nil })
	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				sessionLog.Error("readPump", zap.String("session.id", s.ID), zap.Error(err))
			}
			break
		}

		if err := s.process(message); err != nil {
			sessionLog.Error("process error", zap.String("session.id", s.ID), zap.Error(err))
			break
		}
	}
}

func (s *Session) writePump() {
	ticker := time.NewTicker(s.config.pingPeriod)
	defer func() {
		ticker.Stop()
		_ = s.conn.Close()
		sessionLog.Debug("writePump close", zap.String("session.id", s.ID))
	}()
	for {
		select {
		case message, ok := <-s.send:
			_ = s.conn.SetWriteDeadline(time.Now().Add(s.config.writeWait))
			if !ok {
				// The session closed the channel.
				if err := s.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					// sessionLog.Error("writeCloseMessage error", zap.String("session.id", s.ID), zap.Error(err))
				}
				return
			}

			w, err := s.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				// sessionLog.Error("nextWriter error", zap.String("session.id", s.ID), zap.Error(err))
				return
			}

			if _, err := w.Write(message); err != nil {
				// sessionLog.Error("write error", zap.String("session.id", s.ID), zap.Error(err))
				return
			}

			if err := w.Close(); err != nil {
				// sessionLog.Error("writer close error", zap.String("session.id", s.ID), zap.Error(err))
				return
			}

		case <-ticker.C:
			_ = s.conn.SetWriteDeadline(time.Now().Add(s.config.writeWait))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				sessionLog.Error("ping message error", zap.String("session.id", s.ID), zap.Error(err))
				return
			}
		}
	}
}

func parseRequest(message []byte, response *ResponseMessage) (methodExecutor, error) {
	request := new(RequestMessage)
	if err := json.Unmarshal(message, request); err != nil {
		response.parseError()
		return nil, err
	}

	response.ID = request.ID

	if len(request.Params) == 0 {
		response.invalidParams()
		return nil, fmt.Errorf("invalid params")
	}

	method, err := methodExecutorFactory(*request.Method)
	if err != nil {
		response.methodNotFound()
		return nil, err
	}

	if err := json.Unmarshal(request.Params, &method); err != nil {
		response.parseError()
		return nil, err
	}

	if !method.isValid() {
		response.invalidParams()
		return nil, fmt.Errorf("invalid params")
	}

	return method, nil
}

func (s *Session) process(message []byte) error {
	response := newResponseMessage()
	defer func() {
		raw, err := json.Marshal(response)
		if err != nil {
			sessionLog.Error("response marshal", zap.Error(err))
			return
		}
		s.send <- raw
	}()

	method, err := parseRequest(message, response)
	if err != nil {
		return err
	}

	result, err := method.execute(s)
	if err != nil {
		response.setError(err)

		sessionLog.Debug("response error",
			zap.Stringp("ID", response.ID),
			zap.Int("code", response.Error.Code),
			zap.String("message", response.Error.Message),
		)

		return nil
	}

	if err := response.setResult(result); err != nil {
		response.parseError()
		return err
	}

	sessionLog.Debug("response",
		zap.Stringp("ID", response.ID),
		zap.String("result", string(response.Result)),
	)

	return nil
}
