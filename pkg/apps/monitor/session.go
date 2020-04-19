package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/DaoCasino/platform-action-monitor/pkg/apps/monitor/metrics"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lucsky/cuid"
	"github.com/tevino/abool"
	"go.uber.org/zap"
	"sync"
	"time"
)

type Queue struct {
	flag *abool.AtomicBool
	sync.Mutex
	events []*Event
}

func newQueue() *Queue {
	return &Queue{
		flag:   abool.New(),
		events: make([]*Event, 0),
	}
}

func (q *Queue) open() {
	q.flag.Set()
}

func (q *Queue) isOpen() bool {
	return q.flag.IsSet()
}

func (q *Queue) add(event *Event) {
	q.Lock()
	defer q.Unlock()
	q.events = append(q.events, event)
}

type dataToSocket struct {
	data []byte
	done chan struct{}
	err  error
}

func newSendData(data []byte) *dataToSocket {
	return &dataToSocket{
		data: data,
		done: make(chan struct{}),
		err:  nil,
	}
}

type Session struct {
	ID string

	scraper *Scraper

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send          chan *dataToSocket
	queue         chan *Event
	queueMessages *Queue

	sync.Mutex
	offset uint64
}

func newSession(scraper *Scraper, conn *websocket.Conn) *Session {
	ID := cuid.New()
	sessionLog.Debug("new session", zap.String("ID", ID))

	return &Session{
		ID:            ID,
		scraper:       scraper,
		conn:          conn,
		send:          make(chan *dataToSocket, 512),
		queue:         make(chan *Event),
		queueMessages: newQueue(),
	}
}

func (s *Session) setOffset(offset uint64) {
	s.Lock()
	defer s.Unlock()
	s.offset = offset
}

func (s *Session) Offset() uint64 {
	s.Lock()
	defer s.Unlock()
	return s.offset
}

func (s *Session) readPump(parentContext context.Context) {
	log := sessionLog.Named("readPump")

	readPumpContext, cancel := context.WithCancel(parentContext)
	defer func() {
		cancel()
		sessionManager.unregister <- s
		_ = s.conn.Close()
		log.Debug("pump close", zap.String("session.id", s.ID))
	}()

	log.Debug("pump start", zap.String("session.id", s.ID))

	s.conn.SetReadLimit(config.session.messageSizeLimit)
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

func (s *Session) queuePump(parentContext context.Context, done chan struct{}) {
	log := sessionLog.Named("queuePump")

	defer func() {
		done <- struct{}{}
		close(done)

		log.Debug("pump close", zap.String("session.id", s.ID))
	}()

	log.Debug("pump start", zap.String("session.id", s.ID))

	for {
		select {
		case <-parentContext.Done():
			log.Debug("parent context close", zap.String("session.id", s.ID))
			return
		case event, ok := <-s.queue:
			if !ok {
				return
			}

			if s.queueMessages.isOpen() {
				log.Debug("queue send event", zap.Uint64("event.offset", event.Offset), zap.String("session.id", s.ID))

				events := make([]*Event, 1)
				events[0] = event

				// this blocked
				if err := s.sendChunked(parentContext, events); err != nil {
					log.Debug("sendChunked error", zap.Error(err), zap.String("session.id", s.ID))
					return
				}
			} else {
				log.Debug("queue add event", zap.Uint64("event.offset", event.Offset), zap.String("session.id", s.ID))
				s.queueMessages.add(event)
			}
		}
	}
}

func (s *Session) writePump(parentContext context.Context) {
	log := sessionLog.Named("writePump")
	ticker := time.NewTicker(config.session.pingPeriod)

	ctx, cancel := context.WithCancel(parentContext)
	queuePumpClosed := make(chan struct{})

	defer func() {
		cancel()
		ticker.Stop()
		_ = s.conn.Close()
		log.Debug("pump close", zap.String("session.id", s.ID))
	}()

	log.Debug("pump start", zap.String("session.id", s.ID))
	go s.queuePump(ctx, queuePumpClosed)

	for {
		select {
		case <-parentContext.Done():
			log.Debug("parent context close", zap.String("session.id", s.ID))
			return

		case <-queuePumpClosed:
			if err := sendCloseMessage(s.conn); err != nil {
				log.Error("sendCloseMessage error", zap.Error(err), zap.String("session.id", s.ID))
			}
			return

		case data, ok := <-s.send:
			if !ok {
				log.Debug("send chan close", zap.String("session.id", s.ID))
				if err := sendCloseMessage(s.conn); err != nil {
					log.Error("sendCloseMessage error", zap.Error(err), zap.String("session.id", s.ID))
				}
				return
			}

			if err := sendMessage(s.conn, data); err != nil {
				log.Error("sendMessage error", zap.Error(err), zap.String("session.id", s.ID))
				return
			}
		case <-ticker.C:
			log.Debug("ping", zap.String("session.id", s.ID))

			if err := sendPingMessage(s.conn); err != nil {
				log.Error("sendPingMessage error", zap.Error(err), zap.String("session.id", s.ID))
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

// call from readPump
func (s *Session) process(parentContext context.Context, message []byte) error {
	response := newResponseMessage()
	method, err := parseRequest(message, response)

	defer func() {
		raw, err := json.Marshal(response)
		if err != nil {
			sessionLog.Error("response marshal", zap.Error(err))
			return
		}

		data := newSendData(raw)
		s.send <- data
		<-data.done // TODO: <- block

		if data.err != nil {
			sessionLog.Error("send error", zap.Error(err))
			return
		}

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

// call from readPump it is blocked function
func (s *Session) sendEventsFromDatabase(parentContext context.Context, topic string, offset uint64) error {
	sessionLog.Debug("after subscribe send events", zap.String("session.id", s.ID), zap.Uint64("offset", offset))

	eventType, err := getEventTypeFromTopic(topic)
	if err != nil {
		return fmt.Errorf("get event type error: %s", err)
	}

	var conn *pgxpool.Conn
	conn, err = pool.Acquire(parentContext)
	if err != nil {
		return fmt.Errorf("pool acquire connection error: %s", err)
	}

	defer func() {
		conn.Release()
	}()

	events, err := fetchAllEvents(parentContext, conn.Conn(), offset, 0) // TODO: may be need count
	if err != nil {
		return fmt.Errorf("fetch all events error: %s", err)
	}

	if len(events) == 0 {
		return nil
	}
	filteredEvents := filterEventsByEventType(events, eventType)
	if len(filteredEvents) == 0 {
		return nil
	}

	err = s.sendChunked(parentContext, filteredEvents) // blocked !
	if err != nil {
		return fmt.Errorf("sendChunked error: %s", err)
	}

	return nil
}

func sendPingMessage(conn *websocket.Conn) error {
	if err := conn.SetWriteDeadline(time.Now().Add(config.session.writeWait)); err != nil {
		return fmt.Errorf("SetWriteDeadline error: %s", err)
	}

	if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return fmt.Errorf("writePingMessage error: %s", err)
	}
	return nil
}

func sendCloseMessage(conn *websocket.Conn) error {
	if err := conn.SetWriteDeadline(time.Now().Add(config.session.writeWait)); err != nil {
		return fmt.Errorf("SetWriteDeadline error: %s", err)
	}

	if err := conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
		return fmt.Errorf("writeCloseMessage error: %s", err)
	}
	return nil
}

func sendMessage(conn *websocket.Conn, data *dataToSocket) error {
	data.err = nil

	defer func() {
		data.done <- struct{}{}
		close(data.done)
	}()

	if err := conn.SetWriteDeadline(time.Now().Add(config.session.writeWait)); err != nil {
		data.err = err
		return fmt.Errorf("SetWriteDeadline error: %s", err)
	}

	w, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		data.err = err
		return fmt.Errorf("nextWriter error: %s", err)
	}

	if _, err := w.Write(data.data); err != nil {
		data.err = err
		return fmt.Errorf("write error: %s", err)
	}

	if err := w.Close(); err != nil {
		data.err = err
		return fmt.Errorf("writer close error: %s", err)
	}

	return nil
}

// blocked function, do not call in writePump
func (s *Session) sendChunked(parentContext context.Context, events []*Event) error {
	chunkSize := config.session.maxEventsInMessage
	var offset uint64

loop:
	for i := 0; i < len(events); i += chunkSize {
		end := i + chunkSize

		if end > len(events) {
			end = len(events)
		}

		sendEvents := events[i:end]

		if len(sendEvents) == 0 {
			break
		}

		eventMessage, err := newEventMessage(sendEvents)
		if err != nil {
			return err
		}

		data := newSendData(eventMessage)

		select {
		case <-parentContext.Done():
			sessionLog.Debug("sendChunked parent context done", zap.String("session.id", s.ID))
			break loop
		case s.send <- data:
			<-data.done // TODO: <- block! do not call in writePump

			if data.err != nil {
				return data.err
			}

			offset = sendEvents[len(sendEvents)-1].Offset
			s.setOffset(offset)
			metrics.EventsTotal.Add(float64(len(sendEvents)))
		}
	}

	return nil
}

// this is blocked function!
func (s *Session) sendQueueMessages(parentContext context.Context) error {
	s.queueMessages.Lock()
	defer func() {
		s.queueMessages.events = filterEventsFromOffset(s.queueMessages.events, s.Offset())
		s.queueMessages.Unlock()
	}()

	events := filterEventsFromOffset(s.queueMessages.events, s.Offset())

	if len(events) == 0 {
		return nil
	}

	return s.sendChunked(parentContext, events)
}
