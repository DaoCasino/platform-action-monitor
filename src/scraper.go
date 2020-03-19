package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
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
	message  []byte
	response chan *ScraperResponseMessage
}

type ScraperResponseMessage struct {
	result interface{}
	err    error
}

type Scraper struct {
	abi                *AbiDecoder
	topics             map[string]map[*Session]bool
	unsubscribeSession chan *Session
	subscribe          chan *ScraperSubscribeMessage
	unsubscribe        chan *ScraperUnsubscribeMessage
	broadcast          chan *ScraperBroadcastMessage
}

func newScraper(abi *AbiDecoder) *Scraper {
	return &Scraper{
		abi:                abi,
		topics:             make(map[string]map[*Session]bool),
		subscribe:          make(chan *ScraperSubscribeMessage),
		unsubscribe:        make(chan *ScraperUnsubscribeMessage),
		broadcast:          make(chan *ScraperBroadcastMessage),
		unsubscribeSession: make(chan *Session),
	}
}

func (s *Scraper) run(done <-chan struct{}) {

	defer func() {
		scraperLog.Info("scraper stopped")
	}()

	scraperLog.Info("scraper started")

	for {
		select {
		case <-done:
			return

		case session := <-s.unsubscribeSession:
			for name, topicSessions := range s.topics {
				if _, ok := topicSessions[session]; ok {
					delete(topicSessions, session)
				}

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
				zap.Any("message", json.RawMessage(message.message)),
			)
			response := new(ScraperResponseMessage)

			if topicClients, ok := s.topics[message.name]; ok {
				for clientSession := range topicClients {
					select {
					case clientSession.send <- message.message:
					default:
						s.unsubscribeSession <- clientSession
					}
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

func (s *Scraper) handleNotify(conn *pgx.Conn, offset string, filter *DatabaseFilters) {
	scraperLog.Debug("handleNotify", zap.String("offset", offset))

	data, err := fetchActionData(conn, offset, filter)
	switch err {
	case nil:
		// ok
		if event, err := s.abi.Decode(data); err == nil {
			eventMessage := &EventMessage{offset, event}
			response := newResponseMessage()
			if err := response.setResult(eventMessage); err == nil {
				if raw, err := json.Marshal(response); err == nil {
					select {
					case s.broadcast <- &ScraperBroadcastMessage{"notify", raw, nil}:
					default:
						return
					}
				}
			}
		}
	case pgx.ErrNoRows:
		scraperLog.Debug("no act_data with filter",
			zap.Stringp("act_name", filter.actName),
			zap.Stringp("act_account", filter.actAccount),
		)
	default:
		scraperLog.Error("handleNotify SQL error", zap.Error(err))
		return
	}
}

func (s *Scraper) listen(conn *pgx.Conn, filter *DatabaseFilters, done <-chan struct{}) {
	scraperLog.Debug("listen notify start")

	defer func() {
		scraperLog.Debug("listen notify stop")
	}()

	_, err := conn.Exec(context.Background(), "listen new_action_trace")
	if err != nil {
		scraperLog.Error("error listening new_action_trace", zap.Error(err))
		return
	}

	for {
		select {
		case <-done:
			return
		default:
			notification, err := conn.WaitForNotification(context.Background())
			if err != nil {
				scraperLog.Error("error listening new_action_trace", zap.Error(err))
				return
			}

			scraperLog.Debug("notify",
				zap.Uint32("PID", notification.PID),
				zap.String("channel", notification.Channel),
				zap.String("payload", notification.Payload),
			)

			s.handleNotify(conn, notification.Payload, filter)
		}
	}
}
