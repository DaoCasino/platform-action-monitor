package monitor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMethodExecutorFactory(t *testing.T) {
	_, err := methodExecutorFactory("test_0")
	require.Error(t, err)
}

func TestMethodSubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	subscribe := &methodSubscribeParams{Topic: "test_0"}
	result, err := subscribe.execute(context.Background(), session)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestMethodUnsubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	const topicName = "test_0"

	ctx := context.Background()

	unsubscribe := &methodUnsubscribeParams{Topic: topicName}
	result, err := unsubscribe.execute(ctx, session)
	require.Error(t, err)
	assert.Equal(t, false, result)

	subscribe := &methodSubscribeParams{Topic: topicName}
	_, _ = subscribe.execute(ctx, session)

	result, err = unsubscribe.execute(ctx, session)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestMethodBatchSubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	subscribe := &methodBatchSubscribeParams{Topics: []string{"test_0", "test_1", "test_2"}}
	result, err := subscribe.execute(context.Background(), session)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}

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
