package mywebsocket

import (
	"fmt"
	ws "github.com/gorilla/websocket"
	"sync"
	"time"
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
	return ct, nil
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

func (self *clientManager) startReceiveSendTo() {
	for {
		select {
		case pro := <-self.wantToSendCh:
			if value, ok := self.m.Load(pro.Id); ok {
				if pro.IsJson {
					go func(value any, data any) { value.(Client).WriteJson(data) }(value, pro.Data)
				} else {
					if strMsg, ok := pro.Data.(string); ok {
						go func(value any, data string) { value.(Client).WriteMessage(data) }(value, strMsg)
					}
				}
			}
		}
	}
}
