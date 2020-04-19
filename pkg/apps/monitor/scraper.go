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
		unsubscribeSession: make(chan *Session),
	}
}

func (s *Scraper) run(parentContext context.Context) {
	log := scraperLog.Named("scraper")
	defer func() {
		log.Info("scraper stopped")
	}()
	log.Info("scraper started")

	if pool != nil {
		go scraper.listen(parentContext)
	}

	for {
		select {
		case <-parentContext.Done():
			log.Debug("parent context done")
			return

		case session := <-s.unsubscribeSession:
			log.Debug("unsubscribeSession", zap.String("session.ID", session.ID))
			for name, topicSessions := range s.topics {
				delete(topicSessions, session)

				if len(topicSessions) == 0 {
					delete(s.topics, name)
				}
			}

		case message := <-s.subscribe:
			log.Debug("subscribe",
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
			log.Debug("unsubscribe",
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
			log.Debug("send broadcast",
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

func (s *Scraper) handleNotify(parentContext context.Context, conn *pgx.Conn, offset uint64) error {
	scraperLog.Debug("handleNotify", zap.Uint64("offset", offset))

	s.offset = offset // save current offset
	event, err := fetchEvent(parentContext, conn, offset)

	if err != nil {
		return fmt.Errorf("fetchEvent error: %s", err)
	}

	select {
	case <-parentContext.Done():
	case s.broadcast <- &ScraperBroadcastMessage{fmt.Sprintf("event_%d", event.EventType), event, nil}:
	}
	return nil
}

func (s *Scraper) listen(parentContext context.Context) {
	log := scraperLog.Named("scraper listen")
	conn, err := pool.Acquire(parentContext)
	if err != nil {
		log.Error("pool acquire connection error", zap.Error(err))
	}

	log.Info("listen notify start")

	defer func() {
		conn.Release()
		log.Info("listen notify stop")
	}()

	_, err = conn.Exec(parentContext, "listen new_action_trace")
	if err != nil {
		log.Error("error listening new_action_trace", zap.Error(err))
		return
	}

	for {
		select {
		case <-parentContext.Done():
			log.Debug("listen parent context done")
			return
		default:
			contextWithTimeout, cancelWaitForNotification := context.WithTimeout(parentContext, time.Second)
			notification, err := conn.Conn().WaitForNotification(contextWithTimeout)
			if err == nil {
				log.Debug("notify",
					zap.Uint32("PID", notification.PID),
					zap.String("channel", notification.Channel),
					zap.String("payload", notification.Payload),
				)

				offset, err := strconv.ParseInt(notification.Payload, 10, 64)
				if err != nil {
					log.Error("parseInt error", zap.Error(err))
					cancelWaitForNotification()
					return
				}

				err = s.handleNotify(parentContext, conn.Conn(), uint64(offset))
				if err != nil {
					log.Error("handleNotify error", zap.Error(err))
					cancelWaitForNotification()
					return
				}
			}
			cancelWaitForNotification()
		}
	}
}
