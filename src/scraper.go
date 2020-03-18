package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"strings"
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
	db                 *pgx.Conn
	topics             map[string]map[*Session]bool
	unsubscribeSession chan *Session
	subscribe          chan *ScraperSubscribeMessage
	unsubscribe        chan *ScraperUnsubscribeMessage
	broadcast          chan *ScraperBroadcastMessage
}

func newScraper(db *pgx.Conn) *Scraper {
	return &Scraper{
		db:                 db,
		topics:             make(map[string]map[*Session]bool),
		subscribe:          make(chan *ScraperSubscribeMessage),
		unsubscribe:        make(chan *ScraperUnsubscribeMessage),
		broadcast:          make(chan *ScraperBroadcastMessage),
		unsubscribeSession: make(chan *Session),
	}
}

func (s *Scraper) run(done <-chan struct{}) {
	notification := make(chan string)

	if s.db != nil {
		go listenNotify(s.db, notification)
	}

	defer func() {
		close(notification)
		scraperLog.Info("scraper stopped")
	}()

	scraperLog.Info("scraper started")

	for {
		select {
		case <-done:
			return

		case offset := <-notification:
			scraperLog.Debug("pg notify", zap.String("offset", offset))

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
		1 надо слушать notify pg
		2 надо расарсить запись act_data
		3 надо отправить броадкаст по подписчикам
	*/
}

func listenNotify(db *pgx.Conn, payload chan string) {
	_, err := db.Exec(context.Background(), "listen new_action_trace")
	if err != nil {
		scraperLog.Fatal("error listening new_action_trace", zap.Error(err))
	}

	scraperLog.Debug("listenNotify start")

	defer func() {
		scraperLog.Debug("listenNotify stop")
	}()

	for {
		notification, err := db.WaitForNotification(context.Background())
		if err != nil {
			scraperLog.Error("error listening new_action_trace", zap.Error(err))
			return
		}

		scraperLog.Debug("notify",
			zap.Uint32("PID", notification.PID),
			zap.String("channel", notification.Channel),
			zap.String("payload", notification.Payload),
		)

		select {
		case payload <- notification.Payload:
		default:
			return
		}
	}
}

func getActionData(db *pgx.Conn, offset uint, count uint, filter *DatabaseFilters) ([][]byte, error) {
	var whereParams []string

	if filter != nil {
		if filter.actAccount != nil {
			whereParams = append(whereParams, fmt.Sprintf("act_account='%s'", *filter.actAccount))
		}
		if filter.actName != nil {
			whereParams = append(whereParams, fmt.Sprintf("act_name='%s'", *filter.actName))
		}
	}

	whereParams = append(whereParams, fmt.Sprintf("receipt_global_sequence >= %d", offset))

	var where string
	if len(whereParams) != 0 {
		where = strings.Join(whereParams, " AND ")
		scraperLog.Debug("getActionData", zap.String("where", where))
	}

	sql := fmt.Sprintf("SELECT act_data FROM chain.action_trace WHERE %s ORDER BY receipt_global_sequence ASC LIMIT %d", where, count)
	scraperLog.Debug("getActionData", zap.String("sql", sql))

	rows, _ := db.Query(context.Background(), sql)

	result := make([][]byte, 0)

	for rows.Next() {
		var data []byte
		err := rows.Scan(&data)
		if err != nil {
			return nil, err
		}

		result = append(result, data)
	}

	return result, rows.Err()
}
