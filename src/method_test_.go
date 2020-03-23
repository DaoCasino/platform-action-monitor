package main

/*
import (
	"testing"
)

func TestMethodSubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	subscribe := &methodSubscribeParams{Topic: "test"}
	result, err := subscribe.execute(session)

	if err == nil && result == false {
		t.Error("subscribe method result false; want true")
	}

	if err != nil {
		t.Error(err)
	}
}

func TestMethodUnsubscribe(t *testing.T) {
	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	const topicName = "test"

	unsubscribe := &methodUnsubscribeParams{Topic: topicName}
	result, err := unsubscribe.execute(session)

	if err == nil || result == true {
		t.Error("unsubscribe method error nil or result true; want error message and false")
	}

	subscribe := &methodSubscribeParams{Topic: topicName}
	_, _ = subscribe.execute(session)

	result, err = unsubscribe.execute(session)

	if err == nil && result == false {
		t.Error("unsubscribe method result false; want true")
	}

	if err != nil {
		t.Error(err)
	}
}


*/
