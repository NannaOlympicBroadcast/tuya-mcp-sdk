package mcpsdk

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	StatusNormal = uint32(1)
	StatusStop   = uint32(2)
)

var (
	ErrSessionClosed   = errors.New("session is closed")
	ErrWriteClosed     = errors.New("tried to write to a closed session")
	ErrWriteBufferFull = errors.New("write buffer is full")
)

type envelope struct {
	t   int
	msg []byte
}

// Session wrapper around websocket connections.
type Session struct {
	Request      *http.Request
	Keys         sync.Map
	conn         *websocket.Conn
	input        chan *envelope
	output       chan *envelope
	mcpsdk       *MCPSdk
	status       uint32
	closeOnce    sync.Once
	lastReadTime time.Time
}

func (s *Session) writeMessage(message *envelope) {
	if s.closed() {
		s.mcpsdk.errorHandler(s, ErrWriteClosed)
		return
	}
	defer func() {
		if recover() != nil {
			s.mcpsdk.errorHandler(s, ErrWriteClosed)
		}
	}()
	s.output <- message
}

func (s *Session) writeRaw(message *envelope) error {
	if s.closed() {
		return ErrWriteClosed
	}

	// no error returned from SetWriteDeadline
	_ = s.conn.SetWriteDeadline(time.Now().Add(s.mcpsdk.config.WriteWait))

	err := s.conn.WriteMessage(message.t, message.msg)

	if err != nil {
		return err
	}

	return nil
}

func (s *Session) closed() bool {
	return atomic.LoadUint32(&s.status) == StatusStop
}

func (s *Session) close() {
	s.closeOnce.Do(func() {
		atomic.StoreUint32(&s.status, StatusStop)
		_ = s.conn.Close()
		close(s.output)
	})
}

func (s *Session) writePump(ctx context.Context) {
	ticker := time.NewTicker(s.mcpsdk.config.PingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			println("[Warn::writePump] context is done, stop write pump")
			return
		case msg := <-s.input:
			s.output <- msg
		case msg, ok := <-s.output:
			if !ok {
				return
			}
			err := s.writeRaw(msg)
			if err != nil {
				s.mcpsdk.errorHandler(s, err)
				return
			}
			if msg.t == websocket.CloseMessage {
				return
			}
		case <-ticker.C:
			_ = s.writeRaw(&envelope{t: websocket.PingMessage, msg: []byte{}})
		}
	}
}

func (s *Session) readPump(ctx context.Context) {
	s.conn.SetReadLimit(s.mcpsdk.config.MaxMessageSize)
	s.setReadDeadline()

	s.conn.SetPongHandler(func(string) error {
		s.setReadDeadline()
		s.mcpsdk.pongHandler(s)
		return nil
	})

	if s.mcpsdk.closeHandler != nil {
		s.conn.SetCloseHandler(func(code int, text string) error {
			return s.mcpsdk.closeHandler(s, code, text)
		})
	}

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			println("[Warn::readPump] context is done, stop read pump")
			return
		case <-ticker.C:
			if s.closed() || s.conn == nil {
				println("[Warn::readPump] session is closed or connection is nil, stop read pump")
				return
			}
			t, message, err := s.conn.ReadMessage()
			if err != nil {
				if err == io.EOF {
					println("[Warn::readPump] connection is closed")
					return
				}
				s.mcpsdk.errorHandler(s, err)
				return
			}
			s.setReadDeadline()

			switch t {
			case websocket.TextMessage:
				s.mcpsdk.messageHandler(s, message)
			case websocket.BinaryMessage:
				s.mcpsdk.messageHandlerBinary(s, message)
			}
		}
	}
}

func (s *Session) setReadDeadline() {
	now := time.Now()
	if now.Sub(s.lastReadTime) >= time.Second {
		s.lastReadTime = now
		err := s.conn.SetReadDeadline(s.lastReadTime.Add(s.mcpsdk.config.PongWait + s.mcpsdk.config.PingPeriod))
		if err != nil {
			s.mcpsdk.errorHandler(s, errors.New("failed to set read deadline: "+err.Error()))
		}
	}
}

// Write writes message to session.
func (s *Session) Write(msg []byte) error {
	if s.closed() {
		return ErrSessionClosed
	}

	s.writeMessage(&envelope{t: websocket.TextMessage, msg: msg})
	return nil
}

// WriteBinary writes a binary message to session.
func (s *Session) WriteBinary(msg []byte) error {
	if s.closed() {
		return ErrSessionClosed
	}

	s.writeMessage(&envelope{t: websocket.BinaryMessage, msg: msg})
	return nil
}

// Close closes session.
func (s *Session) Close() error {
	if s.closed() {
		return ErrSessionClosed
	}

	s.writeMessage(&envelope{t: websocket.CloseMessage, msg: []byte{}})
	return nil
}

// Set is used to store a new key/value pair exclusivelly for this session.
// It also lazy initializes s.Keys if it was not used previously.
func (s *Session) Set(key string, value interface{}) {
	s.Keys.Store(key, value)
}

// Get returns the value for the given key, ie: (value, true).
// If the value does not exists it returns (nil, false)
func (s *Session) Get(key string) (value interface{}, exists bool) {
	return s.Keys.Load(key)
}

// MustGet returns the value for the given key if it exists, otherwise it panics.
func (s *Session) MustGet(key string) interface{} {
	if value, exists := s.Get(key); exists {
		return value
	}

	panic("Key \"" + key + "\" does not exist")
}

// UnSet will delete the key and has no return value
func (s *Session) UnSet(key string) {
	s.Keys.Delete(key)
}

// IsClosed returns the status of the connection.
func (s *Session) IsClosed() bool {
	return s.closed()
}
