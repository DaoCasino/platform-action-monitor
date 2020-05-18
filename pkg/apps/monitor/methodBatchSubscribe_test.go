package monitor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMethodBatchSubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	subscribe := &methodBatchSubscribeParams{Topics: []string{"test_0", "test_1", "test_2"}}
	result, err := subscribe.execute(context.Background(), session)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}
