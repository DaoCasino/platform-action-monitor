package monitor

import (
	"fmt"
	"github.com/gorilla/websocket"
	"time"
)

type dataToSocket struct {
	data []byte
	done chan struct{}
	err  error
}

func newSendData(data []byte) *dataToSocket {
	return &dataToSocket{
		data: data,
		done: make(chan struct{}),
		err:  nil,
	}
}

func sendPingMessage(conn *websocket.Conn) error {
	if err := conn.SetWriteDeadline(time.Now().Add(config.session.writeWait)); err != nil {
		return fmt.Errorf("SetWriteDeadline error: %s", err)
	}

	if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return fmt.Errorf("writePingMessage error: %s", err)
	}
	return nil
}

func sendCloseMessage(conn *websocket.Conn) error {
	if err := conn.SetWriteDeadline(time.Now().Add(config.session.writeWait)); err != nil {
		return fmt.Errorf("SetWriteDeadline error: %s", err)
	}

	if err := conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
		return fmt.Errorf("writeCloseMessage error: %s", err)
	}
	return nil
}

func sendMessage(conn *websocket.Conn, data *dataToSocket) error {
	data.err = nil

	defer func() {
		data.done <- struct{}{}
		close(data.done)
	}()

	if err := conn.SetWriteDeadline(time.Now().Add(config.session.writeWait)); err != nil {
		data.err = err
		return fmt.Errorf("SetWriteDeadline error: %s", err)
	}

	w, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		data.err = err
		return fmt.Errorf("nextWriter error: %s", err)
	}

	if _, err := w.Write(data.data); err != nil {
		data.err = err
		return fmt.Errorf("write error: %s", err)
	}

	if err := w.Close(); err != nil {
		data.err = err
		return fmt.Errorf("writer close error: %s", err)
	}

	return nil
}
