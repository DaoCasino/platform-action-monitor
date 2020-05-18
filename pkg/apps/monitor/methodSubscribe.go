package monitor

import (
	"context"
	"go.uber.org/zap"
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

// execute from readPump
func (p *methodSubscribeParams) after(ctx context.Context, session *Session) {
	err := session.sendEventsFromDatabase(ctx, p.Topic, p.Offset) // this block operation
	if err != nil {
		sessionLog.Error("sendEvents error", zap.Error(err), zap.String("session.ID", session.ID))
		return
	}

	sessionLog.Debug("sendEvents done", zap.Uint64("session.offset", session.Offset()), zap.String("session.ID", session.ID))

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
