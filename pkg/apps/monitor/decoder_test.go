package monitor

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type TestEventData struct {
	A uint64 `json:"a"`
	B uint32 `json:"b"`
	C string `json:"c"`
}

func TestDecodeAction(t *testing.T) {
	decoder, err := newDecoder(defaultContractABI)
	require.NoError(t, err)

	data := []byte(`{"data":"","event_type":4,"req_id":3,"game_id":2,"casino_id":1,"sender":"test"}`)

	var encodeBytes, decodeBytes []byte
	encodeBytes, err = decoder.abi.EncodeAction(eos.ActionName(defaultContractActionName), data)
	require.NoError(t, err)

	decodeBytes, err = decoder.decodeAction(encodeBytes, defaultContractActionName)
	require.NoError(t, err)
	assert.Equal(t, data, decodeBytes)
}

func createStructData(t *testing.T, a uint64, b uint32, c string) []byte {
	s := &TestEventData{a, b, c}
	var buffer bytes.Buffer
	encoder := eos.NewEncoder(&buffer)
	err := encoder.Encode(s)
	require.NoError(t, err)

	return buffer.Bytes()
}

func TestDecodeStruct(t *testing.T) {
	decoder, err := newDecoder(defaultEventABI)
	require.NoError(t, err)

	data := createStructData(t, 1, 2, "test_string")

	var decodeBytes []byte
	decodeBytes, err = decoder.decodeStruct(data, defaultEventStructName)
	require.NoError(t, err)

	eventData := new(TestEventData)
	err = json.Unmarshal(decodeBytes, &eventData)
	require.NoError(t, err)

	assert.Equal(t, uint64(1), eventData.A)
	assert.Equal(t, uint32(2), eventData.B)
}

func TestAbiDecoder(t *testing.T) {
	config := newConfig()
	decoder, err := newAbiDecoder(&config.abi)
	require.NoError(t, err)

	data := createStructData(t, 1, 2, "test_string")
	actionJson := fmt.Sprintf(`{"sender":"test","casino_id":"6842030671102619503","game_id":"8251219155248204394","req_id":"5169748975361709968","event_type":0,"data":"%s"}`,
		hex.EncodeToString(data))

	encodeBytes, err := decoder.main.abi.EncodeAction(eos.ActionName(defaultContractActionName), []byte(actionJson))
	require.NoError(t, err)

	var event *Event
	event, _ = decoder.Decode(encodeBytes)
	require.NoError(t, err)
	assert.Equal(t, "test", event.Sender)
	assert.Equal(t, uint64(6842030671102619503), event.CasinoID)
	assert.Equal(t, uint64(8251219155248204394), event.GameID)
	assert.Equal(t, uint64(5169748975361709968), event.RequestID)
	assert.Equal(t, 0, event.EventType)
}
