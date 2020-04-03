package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/DaoCasino/platform-action-monitor/src/metrics"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lucsky/cuid"
	"github.com/tevino/abool"
	"go.uber.org/zap"
	"time"
)

// Count of events in one message, affects the count of allocated memory
const maxEventsInMessage = 100 // TODO: in config?

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
	q.events = q.events[:0] // TODO: cap() save
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

func (s *Session) readPump(parentContext context.Context) {
	readPumpContext, cancel := context.WithCancel(parentContext)
	defer func() {
		cancel()
		sessionManager.unregister <- s
		if err := s.conn.Close(); err != nil {
			sessionLog.Error("readPump connection close", zap.String("session.id", s.ID), zap.Error(err))
		}

		sessionLog.Debug("readPump close", zap.String("session.id", s.ID))
	}()

	sessionLog.Debug("readPump start", zap.String("session.id", s.ID))

	s.conn.SetReadLimit(config.session.maxMessageSize)
	if err := s.conn.SetReadDeadline(time.Now().Add(config.session.pongWait)); err != nil {
		return
	}
	s.conn.SetPongHandler(func(string) error { return s.conn.SetReadDeadline(time.Now().Add(config.session.pongWait)) })

	for {
		select {
		case <-parentContext.Done():
			sessionLog.Debug("readPump parent context close, close connection")
			return
		default:
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					sessionLog.Error("readPump", zap.String("session.id", s.ID), zap.Error(err))
				}
				return
			}

			if err := s.process(readPumpContext, message); err != nil {
				sessionLog.Error("process error", zap.String("session.id", s.ID), zap.Error(err))
				return
			}
		}
	}
}

func (s *Session) queuePump(parentContext context.Context) {
	queuePumpContext, cancel := context.WithCancel(parentContext)
	defer func() {
		cancel()
		s.queueMessages.clean()

		close(s.send)

		sessionLog.Debug("queuePump close", zap.String("session.id", s.ID))
	}()

	sessionLog.Debug("queuePump start", zap.String("session.id", s.ID))

	for {
		select {
		case <-parentContext.Done():
			sessionLog.Debug("queuePump parent context close")
			return
		case event, ok := <-s.queue:
			if !ok {
				return
			}
			s.queueMessages.add(event)
			if s.queueMessages.isOpen() {
				s.sendQueueMessages(queuePumpContext)
			}
		}
	}
}

func (s *Session) writePump(parentContext context.Context) {
	ticker := time.NewTicker(config.session.pingPeriod)
	writePumpContext, cancel := context.WithCancel(parentContext)

	defer func() {
		cancel()
		ticker.Stop()

		if err := s.conn.Close(); err != nil {
			sessionLog.Error("writePump connection close", zap.Error(err), zap.String("session.id", s.ID))
		}

		sessionLog.Debug("writePump close", zap.String("session.id", s.ID))
	}()

	sessionLog.Debug("writePump start", zap.String("session.id", s.ID))
	go s.queuePump(writePumpContext)

	for {
		select {
		case <-parentContext.Done():
			sessionLog.Debug("writePump parent context close")
			return

		case message, ok := <-s.send:
			if err := s.conn.SetWriteDeadline(time.Now().Add(config.session.writeWait)); err != nil {
				sessionLog.Error("SetWriteDeadline error", zap.String("session.id", s.ID), zap.Error(err))
				return
			}

			if !ok {
				// The session closed the channel.
				if err := s.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					sessionLog.Error("writeCloseMessage error", zap.String("session.id", s.ID), zap.Error(err))
				}
				return
			}

			w, err := s.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				sessionLog.Error("nextWriter error", zap.String("session.id", s.ID), zap.Error(err))
				return
			}

			if _, err := w.Write(message); err != nil {
				sessionLog.Error("write error", zap.String("session.id", s.ID), zap.Error(err))
				return
			}

			if err := w.Close(); err != nil {
				sessionLog.Error("writer close error", zap.String("session.id", s.ID), zap.Error(err))
				return
			}

		case <-ticker.C:
			if err := s.conn.SetWriteDeadline(time.Now().Add(config.session.writeWait)); err != nil {
				sessionLog.Error("SetWriteDeadline error", zap.String("session.id", s.ID), zap.Error(err))
				return
			}
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

func (s *Session) process(parentContext context.Context, message []byte) error {
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
			method.after(parentContext, s)
		}
	}()

	if err != nil {
		return err
	}

	result, err := method.execute(parentContext, s)
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

func (s *Session) sendMessages(parentContext context.Context, topic string, offset uint64) {
	sessionLog.Debug("after subscribe send events", zap.String("session.id", s.ID), zap.Uint64("offset", offset))

	eventType, err := getEventTypeFromTopic(topic)
	if err != nil {
		sessionLog.Error("error get event type", zap.String("topic", topic), zap.Error(err))
		return
	}

	// Select current offset
	msg := &ScraperGetOffsetMessage{
		response: make(chan *ScraperResponseMessage),
	}
	s.scraper.getOffset <- msg
	response := <-msg.response

	if response.err != nil {
		sessionLog.Error("get last offset error", zap.Error(err))
		return
	}

	lastOffset := int64(response.result.(uint64) - offset)
	sessionLog.Debug("", zap.Uint64("scraper offset", response.result.(uint64)), zap.Int64("client max offset", lastOffset))

	var conn *pgxpool.Conn
	conn, err = pool.Acquire(parentContext)
	if err != nil {
		sessionLog.Error("pool acquire connection error", zap.Error(err))
		return
	}

	defer func() {
		conn.Release()
		s.sendQueueMessages(parentContext)
		s.queueMessages.open() // TODO: <- check done?
	}()

	for ; lastOffset > 0; lastOffset -= maxEventsInMessage {
		var count uint
		if lastOffset-maxEventsInMessage < 0 {
			count = 0
		} else {
			count = maxEventsInMessage
		}

		// sessionLog.Debug("fetchAllEvents", zap.Uint64("offset", offset), zap.Uint("count", count), zap.Int64("lastOffset", lastOffset))

		events, err := fetchAllEvents(parentContext, conn.Conn(), offset, count)
		if err != nil {
			sessionLog.Error("fetch all events error", zap.Error(err))
			return
		}

		if len(events) == 0 {
			break
		}

		filteredEvents := filterEventsByEventType(events, eventType)

		if len(filteredEvents) > 0 {
			eventMessage, err := newEventMessage(filteredEvents)
			if err != nil {
				sessionLog.Error("error create eventMessage", zap.Error(err))
				return
			}

			select {
			case <-parentContext.Done():
				sessionLog.Debug("sendMessages parent context done")
				return
			case s.send <- eventMessage:
				metrics.EventsTotal.Add(float64(len(filteredEvents)))
			default:
				sessionLog.Error("error send eventMessage")
				return
			}
		}

		s.setOffset(events[len(events)-1].Offset)
		offset = s.offset + 1
	}
}

func (s *Session) sendQueueMessages(parentContext context.Context) {
	sessionLog.Debug("sendQueueMessages", zap.String("session.id", s.ID))

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
	case <-parentContext.Done():
		sessionLog.Debug("sendQueueMessages parent context done", zap.String("session.id", s.ID))
	case s.send <- eventMessage:
		metrics.EventsTotal.Add(float64(len(events)))
	default:
		sessionLog.Error("error send eventMessage")
	}
}
