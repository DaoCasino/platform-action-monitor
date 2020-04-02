package main

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFetchActionData(t *testing.T) {
	config := newConfig()
	mock := &DatabaseMock{}

	testFilter := "test"
	config.db.filter.actName = &testFilter
	config.db.filter.actAccount = &testFilter

	_, err := fetchActionData(context.Background(), mock, 0, &config.db.filter)
	switch err {
	case pgx.ErrNoRows:
	default:
		t.Errorf("error %+v; want ErrNoRows", err)
	}
}

func TestFetchAllActionData(t *testing.T) {
	config := newConfig()

	mock := &DatabaseMock{}
	var result []*ActionTraceRows

	testFilter := "test"
	config.db.filter.actName = &testFilter
	config.db.filter.actAccount = &testFilter

	result, _ = fetchAllActionData(context.Background(), mock, 0, 1, &config.eventExpires, &config.db.filter)
	// require.NoError(t, err)
	assert.Equal(t, len(result), 0)
}
