package monitor

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type EventDataSlice []byte

func (m *EventDataSlice) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(*m))
}

func (m *EventDataSlice) UnmarshalJSON(data []byte) error {
	str := ""
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}

	b, err := hex.DecodeString(str)
	if err != nil {
		return err
	}

	*m = b
	return nil
}

type RawEvent struct {
	Offset    uint64         `json:"offset"`
	Sender    string         `json:"sender"`
	CasinoID  interface{}    `json:"casino_id"`
	GameID    interface{}    `json:"game_id"`
	RequestID interface{}    `json:"req_id"`
	EventType int            `json:"event_type"`
	Data      EventDataSlice `json:"data"`
}

type Event struct {
	Offset    uint64          `json:"offset"`
	Sender    string          `json:"sender"`
	CasinoID  uint64          `json:"casino_id"`
	GameID    uint64          `json:"game_id"`
	RequestID uint64          `json:"req_id"`
	EventType int             `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

func conv(name string, v interface{}) (uint64, error) {
	var result uint64
	var err error

	switch v := v.(type) {
	case float64:
		result = uint64(v)
	case string:
		result, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			return result, err
		}
	default:
		return result, fmt.Errorf("%s unknown type: %T", name, v)
	}

	return result, err
}

func (src *RawEvent) ToEvent(data json.RawMessage) (*Event, error) {
	var err error

	dst := new(Event)
	dst.Offset = src.Offset
	dst.Sender = src.Sender

	dst.CasinoID, err = conv("CasinoID", src.CasinoID)
	if err != nil {
		return nil, err
	}

	dst.GameID, err = conv("GameID", src.GameID)
	if err != nil {
		return nil, err
	}

	dst.RequestID, err = conv("RequestID", src.RequestID)
	if err != nil {
		return nil, err
	}

	dst.EventType = src.EventType

	if len(data) > 0 {
		dst.Data = data
	}

	return dst, nil
}

func newRawEvent(data []byte) (*RawEvent, error) {
	fields := new(RawEvent)
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
	result := events[:0]
	for _, event := range events {
		event := event
		if event.EventType == eventType {
			result = append(result, event)
		}
	}
	return result
}

func filterEventsByEventTypes(events []*Event, eventTypes []int) []*Event {
	result := events[:0]
	for _, event := range events {
		event := event
		for _, eventType := range eventTypes {
			if event.EventType == eventType {
				result = append(result, event)
			}
		}
	}
	return result
}

func filterEventsFromOffset(events []*Event, offset uint64) []*Event {
	for index, event := range events {
		event := event
		if event.Offset > offset {
			return events[index:]
		}
	}

	return events[:0]
}
