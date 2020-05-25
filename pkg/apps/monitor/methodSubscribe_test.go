package monitor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMethodSubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	subscribe := &methodSubscribeParams{Token: "123", Topic: "test_0"}
	result, err := subscribe.execute(context.Background(), session)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}
