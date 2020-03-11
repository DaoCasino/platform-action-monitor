package main

import (
	"fmt"
	"go.uber.org/zap"
)

// type methodParams interface{}
type methodResult interface{}
type methodExecutor interface {
	isValid() bool
	execute(session *Session) (methodResult, error)
}

const (
	methodSubscribe   string = "subscribe"
	methodUnsubscribe string = "unsubscribe"
)

type methodSubscribeParams struct {
	Topic  string `json:"topic"`
	Offset int    `json:"offset"`
}

func (p *methodSubscribeParams) isValid() bool { // TODO: offset же не может быть минусовым?
	return len(p.Topic) > 0 && p.Offset >= 0
}

func (p *methodSubscribeParams) execute(session *Session) (methodResult, error) {
	if loggingEnabled {
		methodLog.Debug("> subscribe",
			zap.String("topic", p.Topic),
			zap.Int("offset", p.Offset),
			zap.String("session.id", session.ID))
	}

	message := &ScraperSubscribeMessage{
		name:     p.Topic,
		session:  session,
		response: make(chan *ScraperResponseMessage),
	}

	session.setOffset(p.Offset) // TODO: не факт что тут?

	session.scraper.subscribe <- message
	response := <-message.response

	return response.result, response.err
}

type methodUnsubscribeParams struct {
	Topic string `json:"topic"`
}

func (p *methodUnsubscribeParams) isValid() bool {
	return len(p.Topic) > 0
}

func (p *methodUnsubscribeParams) execute(session *Session) (methodResult, error) {
	if loggingEnabled {
		methodLog.Debug("> unsubscribe", zap.String("topic", p.Topic), zap.String("session.id", session.ID))
	}

	message := &ScraperUnsubscribeMessage{
		name:     p.Topic,
		session:  session,
		response: make(chan *ScraperResponseMessage),
	}

	session.scraper.unsubscribe <- message
	response := <-message.response

	return response.result, response.err
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
