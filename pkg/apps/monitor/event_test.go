package monitor

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewEvent(t *testing.T) {
	eventJson := []byte(`{"sender":"test","casino_id":"684","game_id":825,"req_id":516,"event_type":0,"data":null}`)
	event, err := newRawEvent(eventJson)

	require.NoError(t, err)
	assert.Equal(t, "test", event.Sender)
	assert.Equal(t, "684", event.CasinoID)
	assert.Equal(t, float64(825), event.GameID)
	assert.Equal(t, float64(516), event.RequestID)
	assert.Equal(t, 0, event.EventType)
}

func TestGetEventTypeFromTopic(t *testing.T) {
	cases := []struct {
		request  string
		expected int
	}{
		{"test_1", 1},
		{"test_2_test_3", 3},
	}

	for _, tc := range cases {
		t.Run(tc.request, func(t *testing.T) {
			actual, err := getEventTypeFromTopic(tc.request)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestFilterEventsByEventType(t *testing.T) {
	events := []*Event{{EventType: 0}, {EventType: 1}, {EventType: 0}, {EventType: 3}}
	result := filterEventsByEventType(events, 0)
	assert.Equal(t, 2, len(result))
}

func TestFilterEventsFromOffset(t *testing.T) {
	events := []*Event{{Offset: 1}, {Offset: 2}, {Offset: 3}}
	result := filterEventsFromOffset(events, 2)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, uint64(3), result[0].Offset)
}
