package monitor

import (
	"fmt"
	"os"
	"time"

	"github.com/DaoCasino/platform-action-monitor/pkg/apps/monitor/metrics"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

const (
	defaultContractActionName = "send"
	defaultEventStructName    = "event_data"
)

type Decoder struct {
	abi *eos.ABI
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

func (a *AbiDecoder) decodeEvent(data []byte) (*RawEvent, error) {
	decodeBytes, err := a.main.decodeAction(data, defaultContractActionName)
	if err != nil {
		return nil, err
	}
	return newRawEvent(decodeBytes)
}

func (a *AbiDecoder) decodeEventData(event int, data []byte) ([]byte, error) {
	start := time.Now()
	defer func() {
		metrics.EventDataDecodingTimeMs.Observe(time.Since(start).Seconds() * 1000)
	}()
	if _, ok := a.events[event]; !ok {
		return nil, fmt.Errorf("no abi with eventType: %d", event)
	}

	return a.events[event].decodeStruct(data, defaultEventStructName)
}

func (a *AbiDecoder) Decode(data []byte) (*Event, error) {
	raw, err := a.decodeEvent(data)
	if err != nil {
		return nil, err
	}

	decodeBytes, err := a.decodeEventData(raw.EventType, raw.Data)
	if err != nil {
		return nil, err
	}

	return raw.ToEvent(decodeBytes)
}
