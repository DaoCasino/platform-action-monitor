package main

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func setupFetchEventTestCase(t *testing.T) (*FetchEvent, func(t *testing.T)) {
	registry := newRegistry()

	config := newConfig()
	registry.set(serviceConfig, config)

	abi, err := newAbiDecoder(&config.abi)
	if err != nil {
		t.Fatal("abi decoder error")
	}
	registry.set(serviceAbiDecoder, abi)

	conn, err := pgx.Connect(context.Background(), config.db.url)
	if err != nil {
		t.Skip("database off")
	}
	registry.set(serviceDatabase, conn)

	fetchEvent := newFetchEvent(registry)
	registry.set(serviceFetchEvent, fetchEvent)

	return fetchEvent, func(t *testing.T) {
		conn.Close(context.Background())
	}
}

func TestFetchEventFetch(t *testing.T) {
	fetchEvent, teardownTestCase := setupFetchEventTestCase(t)
	defer teardownTestCase(t)

	_, err := fetchEvent.fetch("-1")
	require.Error(t, err)
}

func TestFetchEventFetchAll(t *testing.T) {
	fetchEvent, teardownTestCase := setupFetchEventTestCase(t)
	defer teardownTestCase(t)

	testFilter := "test"
	fetchEvent.filter.actAccount = &testFilter
	fetchEvent.filter.actName = &testFilter

	events, err := fetchEvent.fetchAll("-1", 1)

	require.NoError(t, err)
	assert.Equal(t, len(events), 0)
}
