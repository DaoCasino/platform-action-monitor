package main

import (
	"bytes"
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDecodeAction(t *testing.T) {
	decoder, err := newDecoder(defaultContractABI)
	require.NoError(t, err)

	data := []byte(`{"data":"","event_type":4,"req_id":3,"game_id":2,"casino_id":1,"sender":"test"}`)

	encodeBytes, err := decoder.abi.EncodeAction(eos.ActionName(defaultContractActionName), data)
	require.NoError(t, err)

	decodeBytes, err := decoder.decodeAction(encodeBytes, defaultContractActionName)
	assert.Equal(t, data, decodeBytes)
}

func createStructData(t *testing.T, a uint64, b uint32) []byte {
	s := struct {
		A uint64
		B uint32
	}{
		A: a,
		B: b,
	}

	var buffer bytes.Buffer
	encoder := eos.NewEncoder(&buffer)
	err := encoder.Encode(s)
	require.NoError(t, err)

	return buffer.Bytes()
}

func TestDecodeStruct(t *testing.T) {
	decoder, err := newDecoder(defaultEventABI)
	require.NoError(t, err)

	data := createStructData(t, 1, 2)
	decodeBytes, err := decoder.decodeStruct(data, defaultEventStructName)
	assert.Equal(t, decodeBytes, []byte(`{"b":2,"a":1}`))
}

// TODO: full event act_data in binary ??? event.Data type ???
func TestAbiDecoder(t *testing.T) {
	config := newConfig()
	_, err := newAbiDecoder(&config.abi)
	require.NoError(t, err)
}
