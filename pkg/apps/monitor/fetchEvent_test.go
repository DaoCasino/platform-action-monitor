package monitor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchEventFetch(t *testing.T) {
	platform_action_monitor.config = newConfig()
	db := &DatabaseMock{}

	testFilter := "test"
	platform_action_monitor.config.db.filter.actName = &testFilter
	platform_action_monitor.config.db.filter.actAccount = &testFilter

	_, err := fetchEvent(db, 0)
	require.Error(t, err)
}

func TestFetchEventFetchAll(t *testing.T) {
	platform_action_monitor.config = newConfig()
	db := &DatabaseMock{}

	testFilter := "test"
	platform_action_monitor.config.db.filter.actName = &testFilter
	platform_action_monitor.config.db.filter.actAccount = &testFilter

	events, _ := fetchAllEvents(db, 0, 1)
	assert.Equal(t, len(events), 0)
}
