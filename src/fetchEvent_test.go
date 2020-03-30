package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFetchEventFetch(t *testing.T) {
	config = newConfig()
	db := &DatabaseMock{}

	testFilter := "test"
	config.db.filter.actName = &testFilter
	config.db.filter.actAccount = &testFilter

	_, err := fetchEvent(context.Background(), db, 0)
	require.Error(t, err)
}

func TestFetchEventFetchAll(t *testing.T) {
	config = newConfig()
	db := &DatabaseMock{}

	testFilter := "test"
	config.db.filter.actName = &testFilter
	config.db.filter.actAccount = &testFilter

	events, _ := fetchAllEvents(context.Background(), db, 0, 1)
	assert.Equal(t, len(events), 0)
}
