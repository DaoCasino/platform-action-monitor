package monitor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMethodBatchUnsubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	topics := []string{"test_0", "test_1", "test_2"}

	ctx := context.Background()

	unsubscribe := &methodBatchUnsubscribeParams{Topics: topics}
	result, err := unsubscribe.execute(ctx, session)
	require.Error(t, err)
	assert.Equal(t, false, result)

	subscribe := &methodSubscribeParams{Topic: topics[0]}
	_, _ = subscribe.execute(ctx, session)

	result, err = unsubscribe.execute(ctx, session)
	require.Error(t, err)
	assert.Equal(t, false, result)
}
