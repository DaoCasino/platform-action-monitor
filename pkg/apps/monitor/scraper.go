package monitor

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type ScraperSubscribeMessage struct {
	name     string
	session  *Session
	response chan *ScraperResponseMessage
}

type ScraperGetOffsetMessage struct {
	response chan *ScraperResponseMessage
}

type ScraperUnsubscribeMessage struct {
	name     string
	session  *Session
	response chan *ScraperResponseMessage
}

type ScraperBroadcastMessage struct {
	name     string
	event    *Event
	response chan *ScraperResponseMessage
}

type ScraperResponseMessage struct {
	result interface{}
	err    error
}

type Scraper struct {
	unsubscribeSession chan *Session
	subscribe          chan *ScraperSubscribeMessage
	unsubscribe        chan *ScraperUnsubscribeMessage
	broadcast          chan *ScraperBroadcastMessage
	getOffset          chan *ScraperGetOffsetMessage

	topics map[string]map[*Session]bool
	// last offset processed
	offset uint64
}

func newScraper() *Scraper {
	return &Scraper{
		topics:             make(map[string]map[*Session]bool),
		subscribe:          make(chan *ScraperSubscribeMessage),
		unsubscribe:        make(chan *ScraperUnsubscribeMessage),
		broadcast:          make(chan *ScraperBroadcastMessage),
		getOffset:          make(chan *ScraperGetOffsetMessage),
		unsubscribeSession: make(chan *Session),
	}
}

func (s *Scraper) run(parentContext context.Context) {
	defer func() {
		scraperLog.Info("scraper stopped")
	}()
	scraperLog.Info("scraper started")

	if pool != nil {
		go scraper.listen(parentContext)
	}

	for {
		select {
		case <-parentContext.Done():
			scraperLog.Debug("scraper parent context done")
			return

		case message := <-s.getOffset:
			if message.response != nil {
				response := new(ScraperResponseMessage)
				response.result, response.err = s.getLastOffset(parentContext)
				message.response <- response
				close(message.response)
			}

		case session := <-s.unsubscribeSession:
			for name, topicSessions := range s.topics {
				delete(topicSessions, session)

				if len(topicSessions) == 0 {
					delete(s.topics, name)
				}
			}

		case message := <-s.subscribe:
			scraperLog.Debug("subscribe",
				zap.String("name", message.name),
				zap.String("session.id", message.session.ID),
			)

			if topicClients, ok := s.topics[message.name]; ok {
				topicClients[message.session] = true
			} else {
				topicClients := make(map[*Session]bool)
				topicClients[message.session] = true
				s.topics[message.name] = topicClients
			}

			if message.response != nil {
				response := new(ScraperResponseMessage)
				response.result = true
				message.response <- response
				close(message.response)
			}

		case message := <-s.unsubscribe:
			scraperLog.Debug("unsubscribe",
				zap.String("name", message.name),
				zap.String("session.id", message.session.ID),
			)

			response := new(ScraperResponseMessage)

			if topicClients, ok := s.topics[message.name]; ok {
				delete(topicClients, message.session)
				if len(topicClients) == 0 {
					delete(s.topics, message.name)
				}
				response.result = true
			} else {
				response.result = false
				response.err = fmt.Errorf("topic %s not exist", message.name)
			}

			if message.response != nil {
				message.response <- response
				close(message.response)
			}
		case message := <-s.broadcast:
			scraperLog.Debug("send broadcast",
				zap.String("name", message.name),
			)
			response := new(ScraperResponseMessage)

			if topicClients, ok := s.topics[message.name]; ok {
				for clientSession := range topicClients {
					clientSession.queue <- message.event
				}
				response.result = true
			} else {
				response.result = false
				response.err = fmt.Errorf("topic %s not exist", message.name)
			}

			if message.response != nil {
				message.response <- response
				close(message.response)
			}
		}
	}
}

func (s *Scraper) handleNotify(parentContext context.Context, conn *pgx.Conn, offset uint64) {
	scraperLog.Debug("handleNotify", zap.Uint64("offset", offset))

	if event, err := fetchEvent(parentContext, conn, offset); err == nil {
		s.offset = offset // save current offset

		select {
		case <-parentContext.Done():
			sessionLog.Debug("handleNotify parent context done")
			return
		case s.broadcast <- &ScraperBroadcastMessage{fmt.Sprintf("event_%d", event.EventType), event, nil}:
		default:
			return
		}
	}
}

func (s *Scraper) listen(parentContext context.Context) {
	conn, err := pool.Acquire(parentContext)
	if err != nil {
		scraperLog.Error("pool acquire connection error", zap.Error(err))
	}

	scraperLog.Debug("listen notify start")

	defer func() {
		conn.Release()
		scraperLog.Debug("listen notify stop")
	}()

	_, err = conn.Exec(parentContext, "listen new_action_trace")
	if err != nil {
		scraperLog.Error("error listening new_action_trace", zap.Error(err))
		return
	}

	for {
		select {
		case <-parentContext.Done():
			scraperLog.Debug("listen parent context done")
			return
		default:
			contextWithTimeout, cancelWaitForNotification := context.WithTimeout(parentContext, time.Second)
			notification, err := conn.Conn().WaitForNotification(contextWithTimeout)
			if err == nil {
				scraperLog.Debug("notify",
					zap.Uint32("PID", notification.PID),
					zap.String("channel", notification.Channel),
					zap.String("payload", notification.Payload),
				)

				offset, err := strconv.ParseInt(notification.Payload, 10, 64)
				if err != nil {
					scraperLog.Error("parseInt error", zap.Error(err))
					cancelWaitForNotification()
					return
				}

				s.handleNotify(parentContext, conn.Conn(), uint64(offset)) // TODO: check done?
			}
			cancelWaitForNotification()
		}
	}
}

func (s *Scraper) getLastOffset(parentContext context.Context) (uint64, error) {
	if s.offset != 0 {
		return s.offset, nil
	}

	conn, err := pool.Acquire(parentContext)
	if err != nil {
		scraperLog.Error("pool acquire connection error", zap.Error(err))
		return 0, err
	}

	defer func() {
		conn.Release()
	}()

	err = conn.QueryRow(parentContext, "SELECT max(receipt_global_sequence) AS offset FROM chain.action_trace").Scan(&s.offset)
	return s.offset, err
}
