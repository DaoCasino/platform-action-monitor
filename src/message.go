package main

import "encoding/json"

type RequestMessage struct {
	ID     *string         `json:"id"`
	Method *string         `json:"method"`
	Params json.RawMessage `json:"params"`
}

type ResponseErrorMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ResponseMessage struct {
	ID     *string               `json:"id"`
	Result json.RawMessage       `json:"result"`
	Error  *ResponseErrorMessage `json:"error"`
}

type EventMessage struct {
	Offset string   `json:"offset"` // last event.offset
	Events []*Event `json:"events"`
}

func newEventMessage(events []*Event) ([]byte, error) {
	response := newResponseMessage()
	err := response.setResult(&EventMessage{events[len(events)-1].Offset, events})
	if err != nil {
		return nil, err
	}

	return json.Marshal(response)
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
	response.Error = &ResponseErrorMessage{Code: 0, Message: err.Error()}
}

func (response *ResponseMessage) parseError() {
	response.Error = &ResponseErrorMessage{Code: -32700, Message: "parse error"}
}

func (response *ResponseMessage) methodNotFound() {
	response.Error = &ResponseErrorMessage{Code: -32601, Message: "method not found"}
}

func (response *ResponseMessage) invalidParams() {
	response.Error = &ResponseErrorMessage{Code: -32602, Message: "invalid params"}
}

func newResponseMessage() *ResponseMessage {
	return &ResponseMessage{}
}
