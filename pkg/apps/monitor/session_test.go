package monitor

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"sync"
	"testing"
	"time"
	"unsafe"
)

func setupSessionTestCase(t *testing.T) (*Session, func(t *testing.T)) {
	var err error

	config = newConfig()
	config.skipTokenCheck = true

	scraper = newScraper()
	sessionManager = newSessionManager()

	abiDecoder, err = newAbiDecoder(&config.abi)
	require.NoError(t, err)

	contextSessionTestCase, cancel := context.WithCancel(context.Background())
	go sessionManager.run(contextSessionTestCase)
	t.Log("session manager running")
	go scraper.run(contextSessionTestCase)
	t.Log("scraper running")

	session := newSession(scraper, nil)
	sessionManager.register <- session

	t.Log("session register")

	return session, func(t *testing.T) {
		sessionManager.unregister <- session
		t.Log("session unregister")
		cancel()
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
			`{"id":"4","method":"subscribe","params":{"token":"123","topic":"test","offset":1}}`,
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
		{
			"batch subscribe test",
			`{"id":"4","method":"batchSubscribe","params":{"token":"123","topics":["test"],"offset":1}}`,
			`{"id":"4","result":true,"error":null}`,
		},
		{
			"batch subscribe test invalid params",
			`{"id":"7","method":"batchSubscribe","params":{"topic":""}}`,
			`{"id":"7","result":null,"error":{"code":-32602,"message":"invalid params"}}`,
		},
		{
			"batch unsubscribe test",
			`{"id":"5","method":"batchUnsubscribe","params":{"topics":["test"]}}`,
			`{"id":"5","result":true,"error":null}`,
		},
		{
			"batch unsubscribe error",
			`{"id":"6","method":"batchUnsubscribe","params":{"topics":["sdfsdf"]}}`,
			`{"id":"6","result":null,"error":{"code":0,"message":"topic sdfsdf not exist"}}`,
		},
		{
			"batch unsubscribe invalid params",
			`{"id":"8","method":"batchUnsubscribe","params":{"topics":[]}}`,
			`{"id":"8","result":null,"error":{"code":-32602,"message":"invalid params"}}`,
		},
	}

	session, teardownTestCase := setupSessionTestCase(t)
	defer teardownTestCase(t)

	ctx := context.Background()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			go func(t *testing.T) {
				if err := session.process(ctx, []byte(tc.request)); err != nil {
					t.Log(err)
				}
			}(t)

			result := <-session.send

			result.done <- struct{}{}
			close(result.done)

			if string(result.data) != tc.expected {
				t.Fatalf("expected %s, but got %s", tc.expected, string(result.data))
			}
		})
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrcUnsafe(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func newRandomEvent() *Event {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Event{
		Sender:    RandStringBytesMaskImprSrcUnsafe(6),
		CasinoID:  r.Uint64(),
		GameID:    r.Uint64(),
		RequestID: r.Uint64(),
		EventType: 0,
		Data:      nil,
	}
}

func TestSessionSendQueueMessages(t *testing.T) {
	config = newConfig()
	const numEvents = 10

	session := newSession(nil, nil)
	session.setOffset(0)

	for i := 0; i < numEvents; i++ {
		event := newRandomEvent()
		event.Offset = uint64(i)
		session.queueMessages.add(event)
	}

	assert.Equal(t, numEvents, len(session.queueMessages.events))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for data := range session.send {
			data.done <- struct{}{}
			close(data.done)

			responseMessage := new(ResponseMessage)
			eventMessage := new(EventMessage)

			err := json.Unmarshal(data.data, responseMessage)
			require.NoError(t, err)

			err = json.Unmarshal(responseMessage.Result, &eventMessage)
			require.NoError(t, err)

			assert.Equal(t, numEvents-1, len(eventMessage.Events))

			return
		}
	}()

	err := session.sendQueueMessages(context.Background())
	require.NoError(t, err)
	wg.Wait()
	assert.Equal(t, 0, len(session.queueMessages.events))
}

func TestSessionSendMessages(t *testing.T) {
	const numEvents = 10

	session := newSession(nil, nil)
	session.setOffset(0)

	events := make([]*Event, 0)

	for i := 0; i < numEvents; i++ {
		event := newRandomEvent()
		event.Offset = uint64(i)
		events = append(events, event)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {

		defer wg.Done()
		for data := range session.send {
			responseMessage := new(ResponseMessage)
			eventMessage := new(EventMessage)

			err := json.Unmarshal(data.data, responseMessage)
			require.NoError(t, err)

			err = json.Unmarshal(responseMessage.Result, &eventMessage)
			require.NoError(t, err)

			assert.Equal(t, numEvents, len(eventMessage.Events))
			return
		}
	}()

	eventMessage, err := newEventMessage(events)
	require.NoError(t, err)

	data := newSendData(eventMessage)
	session.send <- data

	wg.Wait()
}
