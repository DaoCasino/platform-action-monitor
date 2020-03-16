package main

import (
	"encoding/json"
	"fmt"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
	"os"
)

const (
	defaultContractActionName = "send"
	defaultEventStructName    = "event"
)

type Decoder struct {
	abi *eos.ABI
}

type Event struct {
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

func (d *Decoder) decodeStruct(data []byte, structName string) ([]byte, error) {
	bytes, err := d.abi.Decode(eos.NewDecoder(data), structName)
	if err != nil {
		decoderLog.Error("decoder struct", zap.String("struct", structName), zap.Error(err))
		return nil, err
	}

	return bytes, nil
}

func newEvent(data []byte) (*Event, error) {
	fields := new(Event)
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

func (a *AbiDecoder) decodeEvent(data []byte) (*Event, error) {
	decodeBytes, err := a.main.decodeAction(data, defaultContractActionName)
	if err != nil {
		return nil, err
	}
	return newEvent(decodeBytes)
}

func (a *AbiDecoder) decodeEventData(event int, data []byte) ([]byte, error) {
	if _, ok := a.events[event]; !ok {
		return nil, fmt.Errorf("no abi with eventType: %d", event)
	}

	return a.events[event].decodeStruct(data, defaultEventStructName)
}

func (a *AbiDecoder) decode(data []byte) ([]byte, error) {
	event, err := a.decodeEvent(data)
	if err != nil {
		return nil, err
	}
	decodeBytes, err := a.decodeEventData(event.EventType, event.Data)
	if err != nil {
		return nil, err
	}
	event.Data = decodeBytes
	return json.Marshal(event)
}
