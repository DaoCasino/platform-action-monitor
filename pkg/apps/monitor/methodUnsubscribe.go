package monitor

import (
	"context"
	"go.uber.org/zap"
)

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
