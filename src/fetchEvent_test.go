package main

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFetchEventFetch(t *testing.T) {
	config = newConfig()
	db, err := pgx.Connect(context.Background(), config.db.url)
	if err != nil {
		t.Skip("database off")
	}
	defer func() {
		db.Close(context.Background())
	}()

	testFilter := "test"
	config.db.filter.actName = &testFilter
	config.db.filter.actAccount = &testFilter

	_, err = fetchEvent(db, "-1")
	require.Error(t, err)
}

func TestFetchEventFetchAll(t *testing.T) {
	config = newConfig()
	db, err := pgx.Connect(context.Background(), config.db.url)
	if err != nil {
		t.Skip("database off")
	}
	defer func() {
		db.Close(context.Background())
	}()

	testFilter := "test"
	config.db.filter.actName = &testFilter
	config.db.filter.actAccount = &testFilter

	events, err := fetchAllEvents(db, "-1", 1)

	require.NoError(t, err)
	assert.Equal(t, len(events), 0)
}
