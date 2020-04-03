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
	execute(ctx context.Context, session *Session) (methodResult, error)
	after(ctx context.Context, session *Session)
}

const (
	methodSubscribe   string = "subscribe"
	methodUnsubscribe string = "unsubscribe"
)

type methodSubscribeParams struct {
	Topic string `json:"topic"`
	// Count  int    `json:"count"`
	Offset uint64 `json:"offset"`
}

func (p *methodSubscribeParams) isValid() bool {
	return p.Topic != ""
}

func (p *methodSubscribeParams) execute(_ context.Context, session *Session) (methodResult, error) {
	methodLog.Debug("> subscribe",
		zap.String("topic", p.Topic),
		zap.Uint64("offset", p.Offset),
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

func (p *methodSubscribeParams) after(ctx context.Context, session *Session) {
	session.sendMessages(ctx, p.Topic, p.Offset)
}

type methodUnsubscribeParams struct {
	Topic string `json:"topic"`
}

func (p *methodUnsubscribeParams) isValid() bool {
	return p.Topic != ""
}

func (p *methodUnsubscribeParams) execute(_ context.Context, session *Session) (methodResult, error) {
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

func (p *methodUnsubscribeParams) after(_ context.Context, _ *Session) {
	methodLog.Debug("after unsubscribe")
}

func methodExecutorFactory(method string) (methodExecutor, error) {
	var params methodExecutor
	switch method {
	case methodSubscribe:
		params = new(methodSubscribeParams)
	case methodUnsubscribe:
		params = new(methodUnsubscribeParams)
	default:
		return nil, fmt.Errorf("method not found")
	}

	return params, nil
}
