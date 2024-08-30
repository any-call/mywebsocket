package mywebsocket

import (
	"fmt"
	ws "github.com/gorilla/websocket"
	"sync"
	"time"
)

type client struct {
	sync.Mutex
	id          string
	conn        *ws.Conn
	isConnected bool
	closeCh     chan<- string

	readCbFun  ReadCBFun
	stopReadCh chan struct{}

	stopHeartCh chan struct{}
	heartBeat   time.Duration

	readJSON bool // 标识是否以 JSON 方式读取消息
}

func NewClient(conn *ws.Conn, Id string, heartBeat time.Duration,
	readJSON bool, readFn ReadCBFun, closeCh chan<- string) Client {
	c := &client{
		id:          Id,
		conn:        conn,
		isConnected: false,
		closeCh:     closeCh,
		stopReadCh:  make(chan struct{}, 1),
		readCbFun:   readFn,
		stopHeartCh: make(chan struct{}, 1),
		heartBeat:   heartBeat,
		readJSON:    readJSON,
	}

	go c.read()
	go c.heartbeat()
	return c
}

func (self *client) ID() string {
	return self.id
}

func (self *client) WriteMessage(message string) error {
	self.Lock()
	if !self.isConnected {
		self.Unlock()
		return ws.ErrCloseSent
	}

	if self.conn == nil {
		self.isConnected = false
		self.Unlock()
		return fmt.Errorf("conn is nil")
	}

	err := self.conn.WriteMessage(ws.TextMessage, []byte(message))
	self.Unlock()
	if err != nil {
		self.Close()
		return err
	}

	return nil
}

func (self *client) WriteJson(data any) error {
	self.Lock()
	if !self.isConnected {
		self.Unlock()
		return ws.ErrCloseSent
	}

	if self.conn == nil {
		self.isConnected = false
		self.Unlock()
		return fmt.Errorf("conn is nil")
	}

	err := self.conn.WriteJSON(data)
	self.Unlock()
	if err != nil {
		self.Close()
		return err
	}

	return nil
}

func (self *client) WriteJsonByReadCb(data any, fn func(rData []byte) error) error {
	self.Lock()
	if !self.isConnected {
		self.Unlock()
		return ws.ErrCloseSent
	}

	if self.conn == nil {
		self.isConnected = false
		self.Unlock()
		return fmt.Errorf("conn is nil")
	}

	if err := self.conn.WriteJSON(data); err != nil {
		self.Unlock()
		self.Close()
		return err
	}

	if fn != nil {
		_, msg, err := self.conn.ReadMessage()
		self.Unlock()
		if err != nil {
			self.Close()
			return err
		}
		return fn(msg)
	} else {
		self.Unlock()
	}

	return nil
}

func (self *client) IsConnect() bool {
	if self.conn == nil {
		return false
	}

	return self.isConnected
}

// sync read 客户端读
func (self *client) read() {
	defer func() {
		p := recover()
		if p != nil {
			fmt.Println("receive panic:", p)
		}
	}()

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
				err := self.conn.ReadJSON(&msg)
				if err != nil {
					self.Close()
					return
				}

				if self.readCbFun != nil {
					go self.readCbFun(self.id, msg)
				}

			} else {
				_, message, err := self.conn.ReadMessage()
				if err != nil {
					return
				}

				if self.readCbFun != nil {
					go self.readCbFun(self.id, message)
				}
			}
			break

		}
	}
}

func (self *client) heartbeat() {
	ticker := time.NewTicker(self.heartBeat)
	defer func() {
		ticker.Stop()
		p := recover()
		if p != nil {
			fmt.Println("receive panic:", p)
		}
	}()

	for {
		select {
		case <-self.stopHeartCh:
			return
		case <-ticker.C:
			if self.conn == nil {
				return
			}

			err := self.conn.WriteControl(ws.PingMessage, []byte{}, time.Now().Add(time.Second))
			if err != nil {
				self.Close()
				return
			}
			break
		}
	}
}

func (self *client) Close() {
	self.Lock()
	defer self.Unlock()
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
