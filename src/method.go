package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
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

func (p *methodSubscribeParams) after(session *Session) {
	methodLog.Debug("after subscribe send events", zap.String("session.id", session.ID), zap.String("offset", p.Offset))

	conn, err := pool.Acquire(context.Background())
	if err != nil {
		methodLog.Error("pool acquire connection error", zap.Error(err))
	}

	defer func() {
		conn.Release()
	}()

	events, err := fetchAllEvents(conn.Conn(), p.Offset, 0)
	if err != nil {
		methodLog.Error("fetch all events error", zap.Error(err))
		return
	}

	// TODO: надо сделать фильтр по типу топика и посылать только те что надо
	var eventMessage []byte
	eventMessage, err = newEventMessage(events)
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

	session.setOffset(events[len(events)-1].Offset)
	session.queueMessages.open()
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
