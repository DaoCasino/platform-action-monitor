package main

import (
	"encoding/json"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type Event struct {
	Offset    string          `json:"offset"`
	Sender    string          `json:"sender"`
	CasinoID  string          `json:"casino_id"`
	GameID    string          `json:"game_id"`
	RequestID string          `json:"req_id"`
	EventType int             `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// data  - json byte string
func newEvent(data []byte) (*Event, error) {
	fields := new(Event)
	if err := json.Unmarshal(data, fields); err != nil {
		decoderLog.Error("parse contract fields error", zap.Error(err))
		return nil, err
	}

	return fields, nil
}

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
