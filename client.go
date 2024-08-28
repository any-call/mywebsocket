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
	heartBeat   time.Duration

	authFunc func(conn *ws.Conn) error
	readJSON bool // 标识是否以 JSON 方式读取消息
}

func NewClientWithAuth(Id string, url string, heartBeat time.Duration, authFunc func(conn *ws.Conn) error, readJSON bool, closeCh chan<- string) Client {
	return &client{
		id:          Id,
		conn:        nil,
		mu:          &sync.Mutex{},
		url:         url,
		isConnected: false,
		closeCh:     closeCh,
		readCh:      make(chan any, 100),
		stopReadCh:  make(chan struct{}, 1),
		stopHeartCh: make(chan struct{}, 1),
		heartBeat:   heartBeat,
		authFunc:    authFunc,
		readJSON:    readJSON,
	}
}

func NewClient(id string, url string, heartBeat time.Duration, readJSON bool, closeCh chan<- string) Client {
	return NewClientWithAuth(id, url, heartBeat, nil, readJSON, closeCh)
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

func (self *client) WriteJsonByReadCb(data any, fn func(rData []byte) error) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.isConnected {
		return ws.ErrCloseSent
	}

	if err := self.conn.WriteJSON(data); err != nil {
		return err
	}

	if fn != nil {
		_, msg, err := self.conn.ReadMessage()
		if err != nil {
			return err
		}
		return fn(msg)
	}
	return nil
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
			self.mu.Lock()
			if self.conn == nil {
				self.mu.Unlock()
				return
			}

			if self.readJSON {
				var msg any
				err := self.conn.ReadJSON(&msg)
				self.mu.Unlock()
				if err != nil {
					self.Close()
					return
				}
				self.readCh <- msg
			} else {
				_, message, err := self.conn.ReadMessage()
				self.mu.Unlock()
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
			if !self.mu.TryLock() {
				continue
			}

			if self.conn == nil {
				self.mu.Unlock()
				return
			}

			err := self.conn.WriteControl(ws.PingMessage, []byte{}, time.Now().Add(time.Second))
			self.mu.Unlock()
			if err != nil {
				self.Close()
				return
			}
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
	self.stopReadCh <- struct{}{}
	_ = self.conn.Close()
	self.conn = nil

	if self.closeCh != nil {
		self.closeCh <- self.id
	}

}
