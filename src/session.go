package main

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lucsky/cuid"
)

type Queue struct {
	isOpen bool
	events []*Event
}

func newQueue() *Queue {
	return &Queue{false, make([]*Event, 0)}
}

type Session struct {
	ID     string
	offset string

	registry *Registry

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send                  chan []byte
	queue                 chan *Event
	queueMessages         *Queue
	idleOpenQueueMessages chan struct{}
}

//func newSession(config *SessionConfig, scraper *Scraper, manager *SessionManager, conn *websocket.Conn) *Session {

func newSession(registry *Registry, conn *websocket.Conn) *Session {
	ID := cuid.New()
	sessionLog.Debug("new session", zap.String("ID", ID))

	return &Session{
		ID:                    ID,
		registry:              registry,
		conn:                  conn,
		send:                  make(chan []byte, 512),
		queue:                 make(chan *Event, 32),
		queueMessages:         newQueue(),
		idleOpenQueueMessages: make(chan struct{}),
	}
}

func (s *Session) setOffset(offset string) {
	s.offset = offset
}

func (s *Session) readPump() {

	manager := s.registry.get(serviceSessionManager).(*SessionManager)
	config := s.registry.get(serviceConfig).(*Config)

	defer func() {
		manager.unregister <- s
		if err := s.conn.Close(); err != nil {
			// sessionLog.Error("connection close error", zap.String("session.id", s.ID), zap.Error(err))
		}

		sessionLog.Debug("readPump close", zap.String("session.id", s.ID))
	}()

	s.conn.SetReadLimit(config.session.maxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(config.session.pongWait))
	s.conn.SetPongHandler(func(string) error { s.conn.SetReadDeadline(time.Now().Add(config.session.pongWait)); return nil })
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
	config := s.registry.get(serviceConfig).(*Config)

	ticker := time.NewTicker(config.session.pingPeriod)
	defer func() {
		ticker.Stop()
		_ = s.conn.Close()
		sessionLog.Debug("writePump close", zap.String("session.id", s.ID))
	}()
	for {
		select {
		case event := <-s.queue:
			sessionLog.Debug("add event in queue", zap.String("session.id", s.ID), zap.String("event.offset", event.Offset))
			// TODO: !!!! доедлатьяс
			//s.queueMessages.events[event.Offset]=event
			//if s.queueMessages.isOpen {
			//	s.sendQueueMessages()
			//}

		case <-s.idleOpenQueueMessages:
			s.queueMessages.isOpen = true
			s.sendQueueMessages()

		case message, ok := <-s.send:
			_ = s.conn.SetWriteDeadline(time.Now().Add(config.session.writeWait))
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
			_ = s.conn.SetWriteDeadline(time.Now().Add(config.session.writeWait))
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
	method, err := parseRequest(message, response)
	if err != nil {
		return err
	}

	defer func() {
		raw, err := json.Marshal(response)
		if err != nil {
			sessionLog.Error("response marshal", zap.Error(err))
			return
		}
		s.send <- raw

		method.after(s)
	}()

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

func (s *Session) sendQueueMessages() {
	sessionLog.Debug("sendQueueMessages start")

}

func newEventMessage(events []*Event) ([]byte, error) {
	response := newResponseMessage()
	err := response.setResult(&EventMessage{events[len(events)-1].Offset, events})
	if err != nil {
		return nil, err
	}

	return json.Marshal(response)
}

//
//func (s *Session) init(conn *pgx.Conn, filter *DatabaseFilters) error {
//	rows, err := fetchAllActionData(conn, s.offset, 0, filter)
//	if err != nil {
//		return err
//	}
//
//
//}
