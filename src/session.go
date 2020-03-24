package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lucsky/cuid"
	"github.com/tevino/abool"
	"go.uber.org/zap"
	"time"
)

type Queue struct {
	flag   *abool.AtomicBool
	events []*Event
}

func newQueue() *Queue {
	return &Queue{abool.New(), make([]*Event, 0)}
}

func (q *Queue) open() {
	q.flag.Set()
}

func (q *Queue) isOpen() bool {
	return q.flag.IsSet()
}

func (q *Queue) add(event *Event) {
	q.events = append(q.events, event)
}

func (q *Queue) clean() {
	q.events = q.events[:0]
}

type Session struct {
	ID     string
	offset uint64

	scraper *Scraper

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send          chan []byte
	queue         chan *Event
	queueMessages *Queue
}

//func newSession(config *SessionConfig, scraper *Scraper, manager *SessionManager, conn *websocket.Conn) *Session {

func newSession(scraper *Scraper, conn *websocket.Conn) *Session {
	ID := cuid.New()
	sessionLog.Debug("new session", zap.String("ID", ID))

	return &Session{
		ID:            ID,
		scraper:       scraper,
		conn:          conn,
		send:          make(chan []byte, 512),
		queue:         make(chan *Event, 32),
		queueMessages: newQueue(),
	}
}

func (s *Session) setOffset(offset uint64) {
	s.offset = offset
}

func (s *Session) readPump() {

	defer func() {
		sessionManager.unregister <- s
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
	ticker := time.NewTicker(config.session.pingPeriod)
	defer func() {
		ticker.Stop()
		_ = s.conn.Close()
		sessionLog.Debug("writePump close", zap.String("session.id", s.ID))
	}()
	for {
		select {
		case event := <-s.queue:
			sessionLog.Debug("add event in queue", zap.String("session.id", s.ID), zap.Uint64("event.offset", event.Offset))
			s.queueMessages.add(event)
			if s.queueMessages.isOpen() {
				s.sendQueueMessages()
			}

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

	defer func() {
		raw, err := json.Marshal(response)
		if err != nil {
			sessionLog.Error("response marshal", zap.Error(err))
			return
		}
		s.send <- raw

		if method != nil {
			method.after(s)
		}
	}()

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

func (s *Session) sendMessages(topic string, offset uint64) {
	// TODO: нужно замокать базу
	sessionLog.Debug("after subscribe send events", zap.String("session.id", s.ID), zap.Uint64("offset", offset))

	eventType, err := getEventTypeFromTopic(topic)
	if err != nil {
		sessionLog.Error("error get event type", zap.String("topic", topic), zap.Error(err))
		return
	}

	var conn *pgxpool.Conn
	conn, err = pool.Acquire(context.Background())
	if err != nil {
		sessionLog.Error("pool acquire connection error", zap.Error(err))
		return
	}

	defer func() {
		conn.Release()

		s.sendQueueMessages()
		s.queueMessages.open() // TODO: <-
	}()

	var events []*Event
	events, err = fetchAllEvents(conn.Conn(), offset, 0)
	if err != nil {
		sessionLog.Error("fetch all events error", zap.Error(err))
		return
	}

	if len(events) > 0 {
		filteredEvents := filterEventsByEventType(events, eventType)

		if len(filteredEvents) > 0 {
			eventMessage, err := newEventMessage(filteredEvents)
			if err != nil {
				sessionLog.Error("error create eventMessage", zap.Error(err))
				return
			}

			select {
			case s.send <- eventMessage:
			default:
				sessionLog.Error("error send eventMessage")
				return
			}
		}

		s.setOffset(events[len(events)-1].Offset)
	}
}

func (s *Session) sendQueueMessages() {
	sessionLog.Debug("sendQueueMessages")

	events, err := filterEventsFromOffset(s.queueMessages.events, s.offset)
	if err != nil {
		sessionLog.Error("filterEventsFromOffset", zap.Error(err))
	}

	if len(events) == 0 {
		return
	}

	var eventMessage []byte
	eventMessage, err = newEventMessage(events)
	if err != nil {
		sessionLog.Error("error create eventMessage", zap.Error(err))
		return
	}
	s.queueMessages.clean()

	select {
	case s.send <- eventMessage:
	default:
		sessionLog.Error("error send eventMessage")
		return
	}
}
