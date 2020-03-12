package main

import (
	"testing"
)

func setupSessionTestCase(t *testing.T) (*Session, func(t *testing.T)) {
	config := newConfig()
	scraper := newScraper()
	manager := newSessionManager(&config.upgrader)
	done := make(chan struct{})
	go manager.run(done)
	t.Log("session manager running")
	go scraper.run(done)
	t.Log("scraper running")

	session := newSession(&config.session, scraper, manager, nil)
	session.manager.register <- session

	t.Log("session register")

	return session, func(t *testing.T) {
		session.manager.unregister <- session
		t.Log("session unregister")
		close(done)
		t.Log("scraper stopped")
		t.Log("session manager stopped")
	}
}

func TestSessionProcess(t *testing.T) {
	cases := []struct {
		name     string
		request  string
		expected string
	}{
		{
			"parse error",
			"sdfsdfsdf",
			`{"id":null,"result":null,"error":{"code":-32700,"message":"parse error"}}`,
		},
		{
			"method not found",
			`{"id":"2","method":"sdfsdf","params":{"topic":"lol"}}`,
			`{"id":"2","result":null,"error":{"code":-32601,"message":"method not found"}}`,
		},
		{
			"invalid params",
			`{"id":"3","method":"subscribe"}`,
			`{"id":"3","result":null,"error":{"code":-32602,"message":"invalid params"}}`,
		},
		{
			"subscribe test",
			`{"id":"4","method":"subscribe","params":{"topic":"test","offset":1}}`,
			`{"id":"4","result":true,"error":null}`,
		},
		{
			"subscribe test invalid params",
			`{"id":"7","method":"subscribe","params":{"topic":""}}`,
			`{"id":"7","result":null,"error":{"code":-32602,"message":"invalid params"}}`,
		},
		{
			"unsubscribe test",
			`{"id":"5","method":"unsubscribe","params":{"topic":"test"}}`,
			`{"id":"5","result":true,"error":null}`,
		},
		{
			"unsubscribe error",
			`{"id":"6","method":"unsubscribe","params":{"topic":"sdfsdf"}}`,
			`{"id":"6","result":null,"error":{"code":0,"message":"topic sdfsdf not exist"}}`,
		},
		{
			"unsubscribe invalid params",
			`{"id":"8","method":"unsubscribe","params":{"topic":""}}`,
			`{"id":"8","result":null,"error":{"code":-32602,"message":"invalid params"}}`,
		},
	}

	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session.process([]byte(tc.request))

			result := <-session.send
			if string(result) != tc.expected {
				t.Fatalf("expected %s, but got %s", tc.expected, string(result))
			}
		})
	}
}
