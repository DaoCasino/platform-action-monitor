package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

// type methodParams interface{}
type methodResult interface{}
type methodExecutor interface {
	isValid() bool
	execute(session *Session) (methodResult, error)
	after(session *Session)
}

const (
	methodSubscribe   string = "subscribe"
	methodUnsubscribe string = "unsubscribe"
)

type methodSubscribeParams struct {
	Topic string `json:"topic"`
	// Count  int    `json:"count"`
	Offset string `json:"offset"`
}

func (p *methodSubscribeParams) isValid() bool {
	return len(p.Topic) > 0 && len(p.Offset) > 0
}

func (p *methodSubscribeParams) execute(session *Session) (methodResult, error) {
	methodLog.Debug("> subscribe",
		zap.String("topic", p.Topic),
		zap.String("offset", p.Offset),
		// zap.Int("count", p.Count),
		zap.String("session.id", session.ID))

	message := &ScraperSubscribeMessage{
		name:     p.Topic,
		session:  session,
		response: make(chan *ScraperResponseMessage),
	}

	session.setOffset(p.Offset)
	scraper.subscribe <- message
	response := <-message.response
	return response.result, response.err
}

// TODO: может куда-то переместить эти функции
// Topic name event_0
func getEventTypeFromTopic(topic string) (int, error) {
	s := strings.Split(topic, "_")
	return strconv.Atoi(s[len(s)-1])
}

func filterEventsByEventType(events []*Event, eventType int) []*Event {
	result := make([]*Event, 0)
	for _, event := range events {
		if event.EventType == eventType {
			result = append(result, event)
		}
	}
	return result
}

func filterEventsFromOffset(events []*Event, offset string) ([]*Event, error) {
	offsetInt, err := strconv.Atoi(offset) // TODO: можно лучше...
	if err != nil {
		return nil, err
	}

	for index, event := range events {
		off, err := strconv.Atoi(event.Offset)
		if err != nil {
			return nil, err
		}

		if off > offsetInt {
			return events[index:], nil
		}
	}

	return nil, nil
}

func (p *methodSubscribeParams) after(session *Session) {
	methodLog.Debug("after subscribe send events", zap.String("session.id", session.ID), zap.String("offset", p.Offset))

	eventType, err := getEventTypeFromTopic(p.Topic)
	if err != nil {
		methodLog.Error("error get event type", zap.String("topic", p.Topic), zap.Error(err))
		return
	}

	var conn *pgxpool.Conn
	conn, err = pool.Acquire(context.Background())
	if err != nil {
		methodLog.Error("pool acquire connection error", zap.Error(err))
		return
	}

	defer func() {
		conn.Release()
		session.queueMessages.open() // TODO: <-
	}()

	var events []*Event
	events, err = fetchAllEvents(conn.Conn(), p.Offset, 0)
	if err != nil {
		methodLog.Error("fetch all events error", zap.Error(err))
		return
	}

	if len(events) > 0 {
		filteredEvents := filterEventsByEventType(events, eventType)

		if len(filteredEvents) > 0 {
			eventMessage, err := newEventMessage(filteredEvents)
			if err != nil {
				methodLog.Error("error create eventMessage", zap.Error(err))
				return
			}

			select {
			case session.send <- eventMessage:
			default:
				methodLog.Error("error send eventMessage")
				return
			}
		}

		session.setOffset(events[len(events)-1].Offset)
	}
}

type methodUnsubscribeParams struct {
	Topic string `json:"topic"`
}

func (p *methodUnsubscribeParams) isValid() bool {
	return len(p.Topic) > 0
}

func (p *methodUnsubscribeParams) execute(session *Session) (methodResult, error) {
	methodLog.Debug("> unsubscribe", zap.String("topic", p.Topic), zap.String("session.id", session.ID))

	message := &ScraperUnsubscribeMessage{
		name:     p.Topic,
		session:  session,
		response: make(chan *ScraperResponseMessage),
	}

	scraper.unsubscribe <- message
	response := <-message.response

	return response.result, response.err
}

func (p *methodUnsubscribeParams) after(session *Session) {
	methodLog.Debug("after unsubscribe")
}

func methodExecutorFactory(method string) (methodExecutor, error) {
	var params methodExecutor
	switch method {
	case methodSubscribe:
		params = new(methodSubscribeParams)
		break
	case methodUnsubscribe:
		params = new(methodUnsubscribeParams)
		break
	default:
		return nil, fmt.Errorf("method not found")
	}

	return params, nil
}
