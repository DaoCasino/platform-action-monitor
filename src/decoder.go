package main

import (
	"encoding/json"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
	"os"
)

type Decoder struct {
	abi    *eos.ABI
	action string
}

type ContractFields struct {
	Sender    string          `json:"sender"`
	CasinoID  uint64          `json:"casino_id"`
	GameID    uint64          `json:"game_id"`
	RequestID uint64          `json:"req_id"`
	EventType uint32          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type AbiDecoder struct {
	eventContract *Decoder
	events        map[int]*Decoder
}

const (
	eventRequestDeposit = iota
	eventRequestPlatformAction
	eventRequestCasinoAction
	eventGameFinished
)

func newDecoder(filename string, action string) (*Decoder, error) {
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

	return &Decoder{abi, action}, nil
}

func (d *Decoder) decodeAction(data []byte, actionName string) ([]byte, error) {
	bytes, err := d.abi.DecodeAction(data, eos.ActionName(actionName))
	if err != nil {
		decoderLog.Error("decoder action", zap.String("action", actionName), zap.Error(err))
		return nil, err
	}
	return bytes, nil
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
	a.eventContract, e = newDecoder(c.events.file, c.events.action)
	if e != nil {
		return
	}

	a.events = make(map[int]*Decoder)
	a.events[eventRequestDeposit], e = newDecoder(c.reqDeposit.file, c.reqDeposit.action)
	if e != nil {
		return
	}
	a.events[eventRequestPlatformAction], e = newDecoder(c.reqPlatformAction.file, c.reqPlatformAction.action)
	if e != nil {
		return
	}
	a.events[eventRequestCasinoAction], e = newDecoder(c.reqCasinoAction.file, c.reqCasinoAction.action)
	if e != nil {
		return
	}
	a.events[eventGameFinished], e = newDecoder(c.gameFinished.file, c.gameFinished.action)
	if e != nil {
		return
	}

	return
}
