package main

import (
	"encoding/json"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
	"os"
)

type Decoder struct {
	abi *eos.ABI
}

type ContractFields struct {
	Sender    string          `json:"sender"`
	CasinoID  uint64          `json:"casino_id"`
	GameID    uint64          `json:"game_id"`
	RequestID uint64          `json:"req_id"`
	EventType int             `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type AbiDecoder struct {
	main   *Decoder
	events map[int]*Decoder
}

func newDecoder(filename string) (*Decoder, error) {
	f, err := os.Open(filename)
	if err != nil {
		decoderLog.Error("decoder file", zap.String("filename", filename), zap.Error(err))
		return nil, err
	}
	defer f.Close()

	abi, err := eos.NewABI(f)
	if err != nil {
		decoderLog.Error("decoder newABI", zap.String("filename", filename), zap.Error(err))
		return nil, err
	}

	return &Decoder{abi}, nil
}

func (d *Decoder) decodeAction(data []byte, actionName string) ([]byte, error) {
	bytes, err := d.abi.DecodeAction(data, eos.ActionName(actionName))
	if err != nil {
		decoderLog.Error("decoder action", zap.String("action", actionName), zap.Error(err))
		return nil, err
	}
	return bytes, nil
}

func (d *Decoder) decodeEvent(data []byte) (*ContractFields, error) {
	decodeBytes, err := d.decodeAction(data, defaultContractActionName)
	if err != nil {
		return nil, err
	}
	return newContractFields(decodeBytes)
}

func newContractFields(data []byte) (*ContractFields, error) {
	fields := new(ContractFields)
	if err := json.Unmarshal(data, fields); err != nil {
		decoderLog.Error("parse contract fields error", zap.Error(err))
		return nil, err
	}

	return fields, nil
}

func newAbiDecoder(c *AbiConfig) (a *AbiDecoder, e error) {
	a = new(AbiDecoder)
	a.main, e = newDecoder(c.main)
	if e != nil {
		return
	}

	a.events = make(map[int]*Decoder)
	for eventType, contractFileName := range c.events {
		a.events[eventType], e = newDecoder(contractFileName)
		if e != nil {
			return
		}
	}

	return
}
