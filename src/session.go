package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lucsky/cuid"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 4
)

type Session struct {
	ID      string
	offset  int
	scraper *Scraper
	manager *SessionManager

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func newSession(scraper *Scraper, manager *SessionManager, conn *websocket.Conn) *Session {
	return &Session{
		ID:      cuid.New(),
		scraper: scraper,
		manager: manager,
		conn:    conn,
		send:    make(chan []byte, 512)}
}

func (s *Session) setOffset(offset int) { // TODO: нужно ли локать? будут ли гонки
	s.offset = offset
}

func (s *Session) readPump() {
	defer func() {
		s.manager.unregister <- s
		s.conn.Close()
	}()
	s.conn.SetReadLimit(maxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error { s.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		s.process(message)
		//if err != nil {
		//	log.Printf("parse error: %v", err)
		//	// break // TODO: тут ошибка парсинга сообщения от клиента - отключать его или нет?
		//}
	}
}

func (s *Session) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		s.conn.Close()
	}()
	for {
		select {
		case message, ok := <-s.send:
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The session closed the channel.
				s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := s.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// TODO: хз как зарефакторить красиво
func (session *Session) process(message []byte) (e error) {
	request := new(RequestMessage)
	response := newResponseMessage()

	if err := json.Unmarshal(message, request); err != nil {
		response.parseError()
		e = err
	} else {
		response.ID = request.ID

		method, err := methodExecutorFactory(*request.Method)

		if err != nil {
			response.methodNotFound()
		} else {
			if len(request.Params) == 0 {
				response.invalidParams()
			} else {
				err := json.Unmarshal(request.Params, &method)
				if err != nil {
					response.parseError()
					e = err
				} else {
					if method.isValid() {
						result, err := method.execute(session)
						if err != nil {
							response.setError(err)
						} else {
							err := response.setResult(result)
							if err != nil {
								response.parseError()
								e = err
							}
						}
					} else {
						response.invalidParams()
					}
				}
			}
		}
	}

	log.Printf("process reponse %+v", response)
	raw, err := json.Marshal(response)
	if err != nil {
		return err
	}

	// log.Printf("%s", raw)
	session.send <- raw
	return e
}
