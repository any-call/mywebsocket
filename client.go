package mywebsocket

import (
	ws "github.com/gorilla/websocket"
	"sync"
	"time"
)

type client struct {
	id          string
	conn        *ws.Conn
	mu          *sync.Mutex
	url         string
	isConnected bool
	closeCh     chan<- string

	readCh     chan any
	stopReadCh chan struct{}

	stopHeartCh chan struct{}
	heartMu     *sync.Mutex
	heartBeat   time.Duration

	authFunc func(conn *ws.Conn) error
	readJSON bool // 标识是否以 JSON 方式读取消息
}

func NewClientWithAuth(Id string, url string, heartBeat time.Duration, authFunc func(conn *ws.Conn) error, readJSON bool) Client {
	return &client{
		id:          Id,
		conn:        nil,
		mu:          &sync.Mutex{},
		url:         url,
		isConnected: false,
		closeCh:     make(chan string),
		readCh:      make(chan any, 100),
		stopReadCh:  make(chan struct{}),
		stopHeartCh: make(chan struct{}),
		heartMu:     &sync.Mutex{},
		heartBeat:   heartBeat,
		authFunc:    authFunc,
		readJSON:    readJSON,
	}
}

func NewClient(id string, url string, heartBeat time.Duration, readJSON bool) Client {
	return NewClientWithAuth(id, url, heartBeat, nil, readJSON)
}

func (self *client) ID() string {
	return self.id
}

func (self *client) WriteMessage(message string) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.isConnected {
		return ws.ErrCloseSent
	}

	return self.conn.WriteMessage(ws.TextMessage, []byte(message))
}

func (self *client) WriteJson(data any) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.isConnected {
		return ws.ErrCloseSent
	}

	return self.conn.WriteJSON(data)
}

func (self *client) ReadChan() <-chan any {
	return self.readCh
}

func (self *client) IsConnect() bool {
	return self.isConnected
}

func (self *client) Connect() error {
	self.mu.Lock()
	defer self.mu.Unlock()

	if self.isConnected {
		return nil //不要重复联接
	}

	var err error
	self.conn, _, err = ws.DefaultDialer.Dial(self.url, nil)
	if err != nil {
		self.conn = nil
		return err
	}

	// 执行认证回调
	if self.authFunc != nil {
		err = self.authFunc(self.conn)
		if err != nil {
			self.conn = nil
			return err
		}
	}

	self.isConnected = true
	// 启动心跳
	go self.heartbeat()

	// 启动消息读取
	go self.read()

	return nil
}

// sync read 客户端读
func (self *client) read() {
	for {
		select {
		case <-self.stopReadCh:
			return
		default:
			if self.conn == nil {
				return
			}

			if self.readJSON {
				var msg any
				if err := self.conn.ReadJSON(&msg); err != nil {
					self.Close()
					return
				}
				self.readCh <- msg
			} else {
				_, message, err := self.conn.ReadMessage()
				if err != nil {
					self.Close()
					return
				}
				self.readCh <- message
			}

			break

		}
	}
}

func (self *client) heartbeat() {
	ticker := time.NewTicker(self.heartBeat)
	defer ticker.Stop()

	for {
		select {
		case <-self.stopHeartCh:
			return
		case <-ticker.C:
			if self.conn == nil {
				return
			}

			if !self.heartMu.TryLock() {
				continue
			}

			if err := self.conn.WriteControl(ws.PingMessage, []byte{}, time.Now().Add(time.Second*2)); err != nil {
				self.Close()
				self.heartMu.Unlock()
				return
			}
			self.heartMu.Unlock()
			break
		}
	}
}

func (self *client) Close() {
	self.mu.Lock()
	defer self.mu.Unlock()
	if self.conn == nil {
		return
	}

	self.isConnected = false
	self.stopHeartCh <- struct{}{}
	self.closeCh <- self.id

	_ = self.conn.Close()
	self.conn = nil
}
