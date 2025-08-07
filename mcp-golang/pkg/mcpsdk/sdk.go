package mcpsdk

import (
	"context"
	"errors"
	"math"
	mcp "mcp-sdk/pkg/mcpcli"
	"mcp-sdk/pkg/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Status string

const (
	StatusConnected    Status = "connected"
	StatusConnecting   Status = "connecting"
	StatusDisconnected Status = "disconnected"
	StatusKickout      Status = "kickout"
)

// Config WsManger configuration struct.
type Config struct {
	WriteWait         time.Duration // Milliseconds until write times out.
	PongWait          time.Duration // Timeout for waiting on pong.
	PingPeriod        time.Duration // Milliseconds between pings.
	MaxMessageSize    int64         // Maximum size in bytes of a message.
	MessageBufferSize int           // The max amount of messages that can be in a sessions buffer before it starts dropping them.
}

func defaultWsConf() *Config {
	return &Config{
		WriteWait:         60 * time.Second,
		PongWait:          60 * time.Second,
		PingPeriod:        (60 * time.Second * 9) / 10,
		MaxMessageSize:    0,
		MessageBufferSize: 1,
	}
}

type handleMessageFunc func(*Session, []byte)
type handleErrorFunc func(*Session, error)
type handleCloseFunc func(*Session, int, string) error
type handleSessionFunc func(*Session) error

type MCPSdk struct {
	authToken            *AuthToken
	config               *Config
	conn                 *websocket.Conn
	messageHandler       handleMessageFunc
	messageHandlerBinary handleMessageFunc
	errorHandler         handleErrorFunc
	closeHandler         handleCloseFunc
	connectHandler       handleSessionFunc
	disconnectHandler    handleSessionFunc
	pongHandler          handleSessionFunc

	mcpServerEndpoint string
	mcpcli            *mcp.Client

	internalEventChan chan EventType
	rwlock            sync.RWMutex
	status            Status
	stopCtx           context.Context
}

type BridgeOption func(*MCPSdk)

func WithMCPServerEndpoint(mcpServerEndpoint string) BridgeOption {
	return func(b *MCPSdk) {
		b.mcpServerEndpoint = mcpServerEndpoint
	}
}

func WithAccessParams(accessKey, accessSecret, tuyaEndpoint string) BridgeOption {
	return func(b *MCPSdk) {
		b.authToken = NewAuthToken(tuyaEndpoint, accessKey, accessSecret)
	}
}

func NewMCPSdk(options ...BridgeOption) (*MCPSdk, error) {
	b := &MCPSdk{
		mcpServerEndpoint: "",
		config:            defaultWsConf(),
		internalEventChan: make(chan EventType, 1),
		status:            StatusDisconnected,
		rwlock:            sync.RWMutex{},
		stopCtx:           context.Background(),
	}

	for _, option := range options {
		option(b)
	}

	if b.authToken == nil {
		return nil, errors.New("authToken is not set")
	}

	handler := NewMCPSdkHandler()
	b.messageHandler = handler.HandleMessageBinary(b)
	b.messageHandlerBinary = handler.HandleMessageBinary(b)
	b.errorHandler = handler.HandleError()
	b.connectHandler = handler.HandleConnect()
	b.disconnectHandler = handler.HandleDisconnect(b)
	b.pongHandler = handler.HandlePong()
	b.closeHandler = handler.HandleClose()

	return b, nil
}

func (b *MCPSdk) GetMCPClient() *mcp.Client {
	return b.mcpcli
}

func (b *MCPSdk) GetAuthToken() string {
	return b.authToken.Data.Token
}

func (b *MCPSdk) Run() error {
	b.checkStatusTimer()
	utils.Go(b.readEvent)
	return b.reconnect()
}

func (b *MCPSdk) checkStatusTimer() {
	timer := time.NewTimer(time.Minute * 5)
	defer timer.Stop()
	utils.Go(func() {
		for {
			select {
			case <-b.stopCtx.Done():
				return
			case <-timer.C:
				if b.getConnStatus() == StatusDisconnected {
					println("[Warn::SDK] connection status is disconnected, reconnect")
					b.reconnect()
				}
			}
		}
	})
}

func (b *MCPSdk) reconnect() (err error) {
	defer func() {
		if err != nil {
			b.setConnStatus(StatusDisconnected)
		}
	}()

	status := b.getConnStatus()
	if status != StatusDisconnected {
		println("[Warn::reconnect] already " + string(status) + ", no need to reconnect")
		return nil
	}

	if b.conn != nil {
		if err = b.conn.Close(); err != nil {
			println("[Error::reconnect] connection close failed: ", err.Error())
		}
	}

	if b.mcpcli == nil {
		var mcpClient *mcp.Client
		mcpClient, err = mcp.NewClient(b.mcpServerEndpoint)
		if err != nil {
			return err
		}
		b.mcpcli = mcpClient
	}

	b.setConnStatus(StatusConnecting)

	if err = b.autoRegister(); err != nil {
		return err
	}
	if err = b.keepalive(); err != nil {
		return err
	}

	utils.Go(b.listener)
	return nil
}

func (b *MCPSdk) autoRegister() error {
	return b.authToken.Auth()
}

func (b *MCPSdk) keepalive() error {
	endpoint, header, err := b.authToken.ConnectHeader()
	if err != nil {
		return err
	}

	headerMap := http.Header{}
	for key, value := range header {
		headerMap.Add(key, value)
	}

	b.conn, _, err = websocket.DefaultDialer.Dial(endpoint, headerMap)
	if err != nil {
		return err
	}

	return nil
}

func (b *MCPSdk) readEvent() {
	defer func() {
		if r := recover(); r != nil {
			println("[Error::readInternalEvent] recover from panic", r)
		}
	}()

	for {
		select {
		case <-b.stopCtx.Done():
			println("[Warn::readInternalEvent] stopCtx is done, drop event")
			if b.internalEventChan != nil {
				close(b.internalEventChan)
			}
			return
		case event := <-b.internalEventChan:
			switch event {
			case EventTypeMigrate:
				// migrate event will be triggered by disconnect, so disconnect success will be handled by reconnect
				b.disconnect()
			case EventTypeDisconnect:
				// all disconnect event will be handled by reconnect
				b.disconnect()
				if err := utils.RetryWithBackoff(math.MaxInt, 1*time.Second, 120*time.Second, func() error {
					return b.reconnect()
				}); err != nil {
					println("[Error::readInternalEvent] retry failed: ", err)
				}
			case EventTypeKickout:
				// kickout event will be triggered by disconnect, so disconnect success will be handled by reconnect
				b.kickout()
				return
			}
		}
	}
}

func (b *MCPSdk) sendEvent(event EventType) {
	select {
	case b.internalEventChan <- event:
	case <-b.stopCtx.Done():
		println("[Warn::sendInternalEvent] stopCtx is done, drop event")
		return
	default:
		println("[Error::sendInternalEvent] channel is full, drop event")
	}
}

func (b *MCPSdk) getConnStatus() Status {
	b.rwlock.RLock()
	defer b.rwlock.RUnlock()
	return b.status
}

func (b *MCPSdk) setConnStatus(status Status) {
	b.rwlock.Lock()
	defer b.rwlock.Unlock()
	b.status = status
}

func (b *MCPSdk) disconnect() error {
	if b.getConnStatus() == StatusDisconnected {
		return nil
	}

	b.setConnStatus(StatusDisconnected)
	if b.mcpcli != nil {
		b.mcpcli.Close()
		b.mcpcli = nil
	}
	if b.conn != nil {
		if err := b.conn.Close(); err != nil {
			println("[Error::disconnect] connection close failed: ", err)
			return err
		}
		b.conn = nil
	}
	return nil
}

func (b *MCPSdk) kickout() {
	b.setConnStatus(StatusKickout)
	b.stopCtx.Done()

	if b.mcpcli != nil {
		b.mcpcli.Close()
		b.mcpcli = nil
	}
	if b.conn != nil {
		if err := b.conn.Close(); err != nil {
			println("[Error::kickout] connection close failed: ", err)
			return
		}
		b.conn = nil
	}

}

func (b *MCPSdk) listener() {
	session := &Session{
		conn:      b.conn,
		output:    make(chan *envelope, 1024),
		mcpsdk:    b,
		status:    StatusNormal,
		closeOnce: sync.Once{},
	}

	if err := b.connectHandler(session); err != nil {
		println("[Error::start] websocket connect handler failed: ", err)
		b.sendEvent(EventTypeDisconnect)
		return
	}
	b.setConnStatus(StatusConnected)

	// 启动写入和读取监听
	go session.writePump(b.stopCtx)
	session.readPump(b.stopCtx)

	session.close()
	b.disconnectHandler(session)
}

// HandleConnect fires fn when a session connects.
func (m *MCPSdk) HandleConnect(fn func(*Session) error) {
	m.connectHandler = fn
}

// HandleDisconnect fires fn when a session disconnects.
func (m *MCPSdk) HandleDisconnect(fn func(*Session) error) {
	m.disconnectHandler = fn
}

// HandlePong fires fn when a pong is received from a session.
func (m *MCPSdk) HandlePong(fn func(*Session) error) {
	m.pongHandler = fn
}

// HandleMessage fires fn when a text message comes in.
func (m *MCPSdk) HandleMessage(fn func(*Session, []byte)) {
	m.messageHandler = fn
}

// HandleMessageBinary fires fn when a binary message comes in.
func (m *MCPSdk) HandleMessageBinary(fn func(*Session, []byte)) {
	m.messageHandlerBinary = fn
}

// HandleError fires fn when a session has an error.
func (m *MCPSdk) HandleError(fn func(*Session, error)) {
	m.errorHandler = fn
}

// HandleClose sets the handler for close messages received from the session.
// The code argument to h is the received close code or CloseNoStatusReceived
// if the close message is empty. The default close handler sends a close frame
// back to the session.
//
// The application must read the connection to process close messages as
// described in the section on Control Frames above.
//
// The connection read methods return a CloseError when a close frame is
// received. Most applications should handle close messages as part of their
// normal error handling. Applications should only set a close handler when the
// application must perform some action before sending a close frame back to
// the session.
func (m *MCPSdk) HandleClose(fn func(*Session, int, string) error) {
	if fn != nil {
		m.closeHandler = fn
	}
}
