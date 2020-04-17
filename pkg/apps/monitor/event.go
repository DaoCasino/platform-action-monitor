package monitor

import (
	"encoding/hex"
	"encoding/json"
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
	CasinoID  uint64         `json:"casino_id"`
	GameID    uint64         `json:"game_id"`
	RequestID uint64         `json:"req_id"`
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

func (src *RawEvent) ToEvent(data json.RawMessage) (*Event, error) {
	// var err error

	dst := new(Event)
	dst.Offset = src.Offset
	dst.Sender = src.Sender

	// TODO: if int large raw event type string!
	dst.CasinoID = src.CasinoID
	dst.GameID = src.GameID
	dst.RequestID = src.RequestID

	//dst.CasinoID, err = strconv.ParseUint(src.CasinoID, 10, 64)
	//if err != nil {
	//	return nil, err
	//}
	//dst.GameID, err = strconv.ParseUint(src.GameID, 10, 64)
	//if err != nil {
	//	return nil, err
	//}
	//dst.RequestID, err = strconv.ParseUint(src.RequestID, 10, 64)
	//if err != nil {
	//	return nil, err
	//}

	dst.EventType = src.EventType
	dst.Data = data

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

func filterEventsFromOffset(events []*Event, offset uint64) ([]*Event, error) {
	for index, event := range events {
		event := event
		if event.Offset >= offset {
			return events[index:], nil
		}
	}

	return nil, nil
}
