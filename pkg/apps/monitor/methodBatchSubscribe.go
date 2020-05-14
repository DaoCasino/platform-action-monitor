package monitor

import (
	"context"
	"go.uber.org/zap"
)

type methodBatchSubscribeParams struct {
	Topics []string `json:"topics"`
	Offset uint64   `json:"offset"`
}

func (p *methodBatchSubscribeParams) isValid() bool {
	return len(p.Topics) > 0
}

func (p *methodBatchSubscribeParams) execute(_ context.Context, session *Session) (methodResult, error) {
	methodLog.Debug("> batch subscribe",
		zap.Strings("topics", p.Topics),
		zap.Uint64("offset", p.Offset),
		// zap.Int("count", p.Count),
		zap.String("session.id", session.ID))

	scraperResponse := make(chan *ScraperResponseMessage)
	for i, topic := range p.Topics {
		message := &ScraperSubscribeMessage{
			name:    topic,
			session: session,
		}

		if i+1 == len(p.Topics) {
			message.response = scraperResponse
		}

		scraper.subscribe <- message
	}

	response := <-scraperResponse
	return response.result, response.err
}

// execute from readPump
func (p *methodBatchSubscribeParams) after(ctx context.Context, session *Session) {
	err := session.sendBatchEventsFromDatabase(ctx, p.Topics, p.Offset) // this block operation
	if err != nil {
		sessionLog.Error("sendBatchEvents error", zap.Error(err), zap.String("session.ID", session.ID))
		return
	}

	sessionLog.Debug("sendBatchEvents done", zap.Uint64("session.offset", session.Offset()), zap.String("session.ID", session.ID))

	err = session.sendQueueMessages(ctx)
	if err != nil {
		sessionLog.Error("sendQueueMessages error", zap.Error(err), zap.String("session.ID", session.ID))
		return
	}

	sessionLog.Debug("sendQueueMessages done, open queueMessages",
		zap.Int("queue len", len(session.queueMessages.events)),
		zap.Uint64("session.offset", session.Offset()),
		zap.String("session.ID", session.ID),
	)
}
