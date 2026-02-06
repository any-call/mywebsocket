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

func (self *clientManager) Connect(conn *ws.Conn, id string, subTypes []string, remoteIP string) (Client, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	ct := NewClient(conn, id, subTypes, remoteIP, time.Second*10, true, self.readCbFun, self.closeCh)
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

func (self *clientManager) GetClient(id string) (Client, bool) {
	if id == "" {
		return nil, false
	}

	value, ok := self.m.Load(id)
	if !ok {
		return nil, false
	}

	c, ok := value.(Client)
	return c, ok
}

// startReceiveSendTo 只负责消息投递：
// - 投递未命中：直接丢弃
// 不做任何路由或业务判断
func (self *clientManager) startReceiveSendTo() {
	for {
		select {
		case pro := <-self.wantToSendCh:
			// 1️⃣ 精确 ID 投递（最高优先级）
			if pro.ID != "" { // 1️⃣ 精确投递
				if value, ok := self.m.Load(pro.ID); ok {
					self.write(value.(Client), pro)
				}
				// 没找到，直接丢
				continue
			}

			// 2️⃣ 按 Topic 订阅转发
			if len(pro.Topics) > 0 {
				self.m.Range(func(_, value any) bool {
					c := value.(Client)
					if topicMatch(pro.Topics, c.Subscriptions()) {
						self.write(c, pro)
					}
					return true
				})
				continue
			}

			//广播 TO 所有客户端
			self.m.Range(func(_, value any) bool {
				self.write(value.(Client), pro)
				return true
			})
		}
	}
}

func (self *clientManager) write(c Client, pro *Message) {
	if pro.IsJson {
		go c.WriteJson(pro.Data)
		return
	}

	if str, ok := pro.Data.(string); ok {
		go c.WriteMessage(str)
	}
}

func topicMatch(msgTopics []string, subs []string) bool {
	if len(msgTopics) == 0 || len(subs) == 0 {
		return false
	}

	subSet := make(map[string]struct{}, len(subs))
	for _, s := range subs {
		subSet[s] = struct{}{}
	}

	for _, t := range msgTopics {
		if _, ok := subSet[t]; ok {
			return true
		}
	}
	return false
}
