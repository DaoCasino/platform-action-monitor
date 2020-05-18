package monitor

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMethodExecutorFactory(t *testing.T) {
	_, err := methodExecutorFactory("test")
	require.Error(t, err)

	var m methodExecutor
	m, err = methodExecutorFactory(methodSubscribe)
	require.NoError(t, err)
	assert.IsType(t, &methodSubscribeParams{}, m)

	m, err = methodExecutorFactory(methodUnsubscribe)
	require.NoError(t, err)
	assert.IsType(t, &methodUnsubscribeParams{}, m)

	m, err = methodExecutorFactory(methodBatchSubscribe)
	require.NoError(t, err)
	assert.IsType(t, &methodBatchSubscribeParams{}, m)

	m, err = methodExecutorFactory(methodBatchUnsubscribe)
	require.NoError(t, err)
	assert.IsType(t, &methodBatchUnsubscribeParams{}, m)
}
