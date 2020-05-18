package monitor

import (
	"context"
	"go.uber.org/zap"
)

type methodBatchUnsubscribeParams struct {
	Topics []string `json:"topics"`
}

func (p *methodBatchUnsubscribeParams) isValid() bool {
	return len(p.Topics) > 0
}

func (p *methodBatchUnsubscribeParams) execute(_ context.Context, session *Session) (methodResult, error) {
	methodLog.Debug("> batch unsubscribe", zap.Strings("topics", p.Topics), zap.String("session.id", session.ID))

	scraperResponse := make(chan *ScraperResponseMessage)
	for i, topic := range p.Topics {
		message := &ScraperUnsubscribeMessage{
			name:    topic,
			session: session,
		}

		if i+1 == len(p.Topics) {
			message.response = scraperResponse
		}

		scraper.unsubscribe <- message
	}

	response := <-scraperResponse
	return response.result, response.err
}

func (p *methodBatchUnsubscribeParams) after(_ context.Context, _ *Session) {
	methodLog.Debug("after batch unsubscribe")
}
