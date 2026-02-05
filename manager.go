package mywebsocket

import (
	"fmt"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
)

type (
	clientManager struct {
		m  *sync.Map
		mu *sync.Mutex

		closeCh             chan string   //closeCh: the client close channel
		createCh, destoryCh chan<- Client // createCh, destroyCh: the client create and destroy channel
		wantToSendCh        chan *Message
		readCbFun           ReadCBFun
	}
)

func NewClientManager(createCh, destoryCh chan<- Client, readFn ReadCBFun) ClientManager {
	manager := &clientManager{
		m:            &sync.Map{},
		mu:           &sync.Mutex{},
		closeCh:      make(chan string, 10000),
		readCbFun:    readFn,
		createCh:     createCh,
		destoryCh:    destoryCh,
		wantToSendCh: make(chan *Message, 1000),
	}
	go manager.listenClose()
	go manager.startReceiveSendTo()
	return manager
}

func (self *clientManager) Connect(conn *ws.Conn, id string) (Client, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	ct := NewClient(conn, id, time.Second*10, true, self.readCbFun, self.closeCh)
	if _, ok := self.m.Load(ct.ID()); ok {
		return nil, fmt.Errorf("client already exists :%s", ct.ID())
	}

	self.m.Store(ct.ID(), ct)

	// then attach to create
	if ch := self.createCh; ch != nil {
		ch <- ct
	}

	return ct, nil
}

func (self *clientManager) TotalConn() int {
	count := 0
	self.m.Range(func(key, _ interface{}) bool {
		count++
		return true
	})

	return count
}

func (self *clientManager) SendToClient(msg *Message) {
	if msg != nil && msg.Data != nil {
		self.wantToSendCh <- msg
	}
}

func (self *clientManager) listenClose() {
	for {
		select {
		case clientID, ok := <-self.closeCh:
			if !ok {
				return
			}
			if value, ok := self.m.Load(clientID); ok {
				if ct, ok := value.(Client); ok {
					// first to delete
					self.m.Delete(clientID)
					// then attach to destroy
					if ch := self.destoryCh; ch != nil {
						ch <- ct
					}
				}
			}
		}
	}
}

func (self *clientManager) RangeConn(fn func(id string, c Client) bool) {
	self.m.Range(func(key, value any) bool {
		id, _ := key.(string)
		c, _ := value.(Client)
		return fn(id, c)
	})
}

func (self *clientManager) startReceiveSendTo() {
	for {
		select {
		case pro := <-self.wantToSendCh:
			if pro.To != "" { // 1️⃣ 精确投递
				if value, ok := self.m.Load(pro.To); ok {
					if pro.IsJson {
						go func(value any, data any) { value.(Client).WriteJson(data) }(value, pro.Data)
					} else {
						if strMsg, ok := pro.Data.(string); ok {
							go func(value any, data string) { value.(Client).WriteMessage(data) }(value, strMsg)
						}
					}
				}
			} else { // 2️⃣ 广播 TO 所有客户端
				self.m.Range(func(_, value any) bool {
					if pro.IsJson {
						go func(value any, data any) { value.(Client).WriteJson(data) }(value, pro.Data)
					} else {
						if strMsg, ok := pro.Data.(string); ok {
							go func(value any, data string) { value.(Client).WriteMessage(data) }(value, strMsg)
						}
					}
					return true
				})
			}
		}
	}
}
