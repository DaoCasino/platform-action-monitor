package monitor

import (
	"context"
	"fmt"
)

// type methodParams interface{}
type methodResult interface{}
type methodExecutor interface {
	isValid() bool
	execute(ctx context.Context, session *Session) (methodResult, error)
	after(ctx context.Context, session *Session)
}

const (
	methodSubscribe        string = "subscribe"
	methodUnsubscribe      string = "unsubscribe"
	methodBatchSubscribe   string = "batchSubscribe"
	methodBatchUnsubscribe string = "batchUnsubscribe"
)

func methodExecutorFactory(method string) (methodExecutor, error) {
	var params methodExecutor
	switch method {
	case methodSubscribe:
		params = new(methodSubscribeParams)
	case methodUnsubscribe:
		params = new(methodUnsubscribeParams)
	case methodBatchSubscribe:
		params = new(methodBatchSubscribeParams)
	case methodBatchUnsubscribe:
		params = new(methodBatchUnsubscribeParams)
	default:
		return nil, fmt.Errorf("method not found")
	}

	return params, nil
}
