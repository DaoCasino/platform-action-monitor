package monitor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMethodExecutorFactory(t *testing.T) {
	_, err := methodExecutorFactory("test")
	require.Error(t, err)
}

func TestMethodSubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	subscribe := &methodSubscribeParams{Topic: "test"}
	result, err := subscribe.execute(session)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestMethodUnsubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	const topicName = "test"

	unsubscribe := &methodUnsubscribeParams{Topic: topicName}
	result, err := unsubscribe.execute(session)
	require.Error(t, err)
	assert.Equal(t, false, result)

	subscribe := &methodSubscribeParams{Topic: topicName}
	_, _ = subscribe.execute(session)

	result, err = unsubscribe.execute(session)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}
