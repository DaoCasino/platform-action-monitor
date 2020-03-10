package main

import (
	"fmt"
	"log"
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
	Topic string `json:"topic"`
	Offset int `json:"offset"`
}

func (p *methodSubscribeParams) isValid() bool { // TODO: offset же не может быть минусовым?
	return len(p.Topic) > 0 && p.Offset >= 0
}

func (p *methodSubscribeParams) execute(session *Session) (methodResult, error) {
	log.Printf("> subscribe topic %s, offset %d; from %s\n", p.Topic, p.Offset, session.ID)

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
	log.Printf("> unsubscribe topic %s, from %s\n", p.Topic, session.ID)

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