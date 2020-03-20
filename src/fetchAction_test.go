package main

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFetchActionData(t *testing.T) {
	config := newConfig()
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

	_, err = fetchActionData(db, "0", &config.db.filter)
	switch err {
	case pgx.ErrNoRows:
	default:
		t.Errorf("error %+v; want ErrNoRows", err)
	}
}

func TestFetchAllActionData(t *testing.T) {
	config := newConfig()
	db, err := pgx.Connect(context.Background(), config.db.url)
	if err != nil {
		t.Skip("database off")
	}
	defer func() {
		db.Close(context.Background())
	}()

	var result []*ActionTraceRows

	testFilter := "test"
	config.db.filter.actName = &testFilter
	config.db.filter.actAccount = &testFilter

	result, err = fetchAllActionData(db, "0", 1, &config.db.filter)
	require.NoError(t, err)
	assert.Equal(t, len(result), 0)
}
