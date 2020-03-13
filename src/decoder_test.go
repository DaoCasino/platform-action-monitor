package main

import (
	"github.com/eoscanada/eos-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDecoder(t *testing.T) {
	decoder, err := newDecoder(defaultContractABI)
	require.NoError(t, err)

	data := []byte(`{"data":"","event_type":4,"req_id":3,"game_id":2,"casino_id":1,"sender":"test"}`)

	encodeBytes, err := decoder.abi.EncodeAction(eos.ActionName(defaultContractActionName), data)
	require.NoError(t, err)

	decodeBytes, err := decoder.decodeAction(encodeBytes, defaultContractActionName)
	assert.Equal(t, data, decodeBytes)

	fields, err := newContractFields(decodeBytes)
	require.NoError(t, err)
	assert.Equal(t, fields.EventType, 4)
}
