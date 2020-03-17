package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"log"
	"strings"
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
	message  []byte
	response chan *ScraperResponseMessage
}

type ScraperResponseMessage struct {
	result interface{}
	err    error
}

type Scraper struct {
	db          *pgx.Conn
	queryPeriod time.Duration

	topics             map[string]map[*Session]bool
	unsubscribeSession chan *Session
	subscribe          chan *ScraperSubscribeMessage
	unsubscribe        chan *ScraperUnsubscribeMessage
	broadcast          chan *ScraperBroadcastMessage
}

func newScraper(db *pgx.Conn, queryPeriod time.Duration) *Scraper {
	return &Scraper{
		db:                 db,
		queryPeriod:        queryPeriod,
		topics:             make(map[string]map[*Session]bool),
		subscribe:          make(chan *ScraperSubscribeMessage),
		unsubscribe:        make(chan *ScraperUnsubscribeMessage),
		broadcast:          make(chan *ScraperBroadcastMessage),
		unsubscribeSession: make(chan *Session),
	}
}

func (s *Scraper) run(done <-chan struct{}) {
	// log.Fatalf("%d", s.queryPeriod)
	ticker := time.NewTicker(s.queryPeriod)

	defer func() {
		ticker.Stop()
		scraperLog.Info("scraper stopped")
	}()

	scraperLog.Info("scraper started")

	for {
		select {
		case <-done:
			return

		case <-ticker.C:
			scraperLog.Debug("timer tick tick")

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
			scraperLog.Debug("broadcast",
				zap.String("name", message.name),
				zap.Binary("message", message.message),
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

func (s *Scraper) process() {
	/*
		1 надо получить запись из бд
		2 надо расарсить запись act_data
		3 надо отправить броадкаст по подписчикам
	*/
}

// TODO: возвращает канал или массив ...
func getActionData(db *pgx.Conn, offset uint, filter *DatabaseFilters) error {
	var whereParams []string
	if filter != nil {
		if filter.actAccount != nil {
			whereParams = append(whereParams, fmt.Sprintf("act_account='%s'", *filter.actAccount))
		}
		if filter.actName != nil {
			whereParams = append(whereParams, fmt.Sprintf("act_name='%s'", *filter.actName))
		}
	}

	var where string
	if len(whereParams) != 0 {
		where = fmt.Sprintf(" WHERE %s", strings.Join(whereParams, " AND "))
		scraperLog.Debug("getActionData", zap.String("where", where))
	}

	rows, _ := db.Query(context.Background(), "SELECT act_data FROM chain.action_trace $1 receipt_global_sequence > $2  ORDER BY receipt_global_sequence ASC", where, offset)

	for rows.Next() {
		var data []byte
		err := rows.Scan(&data)
		if err != nil {
			return err
		}

		log.Printf("%+v", data)
	}

	return rows.Err()
}
