package main

import "encoding/json"

type RequestMessage struct {
	ID       string          `json:"id"`
	Method   string          `json:"method"`
	Params   json.RawMessage `json:"params"`
}

type ResponseMessage struct {
	ID       string          `json:"id"`
	Result   json.RawMessage `json:"result"`
	Error    string          `json:"error"`
}

func (response *ResponseMessage) setResult(data interface{}) error {
	byte1, err := json.Marshal(data)
	if err == nil {
		raw := json.RawMessage(byte1)
		response.Result = raw
		return nil
	}

	return err
}

func (response *ResponseMessage) setError(err error) {
	response.Error = err.Error()
}

func newResponseMessage(id string) *ResponseMessage {
	res := &ResponseMessage {
		ID: id,
	}

	return res
}