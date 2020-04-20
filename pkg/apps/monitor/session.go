package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
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
			sessionLog.Error("send error", zap.Error(data.err))
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
