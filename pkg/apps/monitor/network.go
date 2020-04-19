package monitor

import (
	"context"
	"fmt"
	"github.com/DaoCasino/platform-action-monitor/pkg/apps/monitor/metrics"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

// call from readPump it is blocked function
func (s *Session) sendEventsFromDatabase(parentContext context.Context, topic string, offset uint64) error {
	sessionLog.Debug("after subscribe send events", zap.String("session.id", s.ID), zap.Uint64("offset", offset))

	eventType, err := getEventTypeFromTopic(topic)
	if err != nil {
		return fmt.Errorf("get event type error: %s", err)
	}

	var conn *pgxpool.Conn
	conn, err = pool.Acquire(parentContext)
	if err != nil {
		return fmt.Errorf("pool acquire connection error: %s", err)
	}

	defer func() {
		conn.Release()
	}()

	events, err := fetchAllEvents(parentContext, conn.Conn(), offset, 0) // TODO: may be need count
	if err != nil {
		return fmt.Errorf("fetch all events error: %s", err)
	}

	if len(events) == 0 {
		return nil
	}
	filteredEvents := filterEventsByEventType(events, eventType)
	if len(filteredEvents) == 0 {
		return nil
	}

	err = s.sendChunked(parentContext, filteredEvents) // blocked !
	if err != nil {
		return fmt.Errorf("sendChunked error: %s", err)
	}

	return nil
}

// blocked function, do not call in writePump
func (s *Session) sendChunked(parentContext context.Context, events []*Event) error {
	chunkSize := config.session.maxEventsInMessage
	var offset uint64

loop:
	for i := 0; i < len(events); i += chunkSize {
		end := i + chunkSize

		if end > len(events) {
			end = len(events)
		}

		sendEvents := events[i:end]

		if len(sendEvents) == 0 {
			break
		}

		eventMessage, err := newEventMessage(sendEvents)
		if err != nil {
			return err
		}

		data := newSendData(eventMessage)

		select {
		case <-parentContext.Done():
			sessionLog.Debug("sendChunked parent context done", zap.String("session.id", s.ID))
			break loop
		case s.send <- data:
			<-data.done // TODO: <- block! do not call in writePump

			if data.err != nil {
				return data.err
			}

			offset = sendEvents[len(sendEvents)-1].Offset
			s.setOffset(offset)
			metrics.EventsTotal.Add(float64(len(sendEvents)))
		}
	}

	return nil
}

// this is blocked function!
func (s *Session) sendQueueMessages(parentContext context.Context) error {
	s.queueMessages.Lock()
	defer func() {
		s.queueMessages.events = filterEventsFromOffset(s.queueMessages.events, s.Offset())
		s.queueMessages.Unlock()
	}()

	events := filterEventsFromOffset(s.queueMessages.events, s.Offset())

	if len(events) == 0 {
		return nil
	}

	return s.sendChunked(parentContext, events)
}
