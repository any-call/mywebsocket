package mywebsocket

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ws "github.com/gorilla/websocket"
)

//建一个只读客户端

type WSReceiver[T any] struct {
	addr string
	id   string
	subs []string

	readCh chan<- T

	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	conn      *ws.Conn
	connected atomic.Bool
	onError   func(error)
	heartBeat time.Duration
}

func NewWSReceiver[T any](
	addr string,
	id string,
	subs []string,
	readCh chan<- T,
	onError func(error),
) *WSReceiver[T] {
	return &WSReceiver[T]{
		addr:    addr,
		id:      id,
		subs:    subs,
		readCh:  readCh,
		onError: onError,
	}
}

func (r *WSReceiver[T]) Start(heartBeat time.Duration, headerCb func(header http.Header)) error {
	r.mu.Lock()
	if r.connected.Load() {
		r.mu.Unlock()
		return nil // 或返回 error，看你约定
	}

	// ✅ 每次 Start 都新建 ctx
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.mu.Unlock()

	header := http.Header{}
	header.Set("X-Client-ID", r.id)
	if len(r.subs) > 0 {
		header.Set("X-Client-Subs", strings.Join(r.subs, ","))
	}
	if headerCb != nil {
		headerCb(header)
	}

	conn, _, err := ws.DefaultDialer.Dial(r.addr, header)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.heartBeat = heartBeat
	r.conn = conn
	r.connected.Store(true)
	r.mu.Unlock()

	// 设置 pong 处理（关键）
	r.conn.SetReadDeadline(time.Now().Add(r.heartBeat * 2))
	r.conn.SetPongHandler(func(string) error {
		r.conn.SetReadDeadline(time.Now().Add(r.heartBeat * 2))
		return nil
	})

	// ⚠️ 建议异步，不要阻塞 Start
	go r.readLoop()
	go r.heartbeat()
	return nil
}

func (r *WSReceiver[T]) Close() {
	r.mu.Lock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.mu.Unlock()

	r.closeConn()
}

func (r *WSReceiver[T]) IsConnected() bool {
	return r.connected.Load()
}

func (r *WSReceiver[T]) readLoop() {
	defer r.closeConn()

	internalCh := make(chan T, 1000)

	// 👉 专门负责往外写
	go func() {
		defer close(internalCh)
		for {
			select {
			case msg := <-internalCh:
				select {
				case r.readCh <- msg:
				case <-r.ctx.Done():
					return
				}
			case <-r.ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			var msg T
			if err := r.conn.ReadJSON(&msg); err != nil {
				r.closeConn()
				if r.onError != nil {
					r.onError(err)
				}
				return
			}

			// ⚠️ 这里永远不阻塞 WS 读
			select {
			case internalCh <- msg:
			default:
				// buffer 满了：丢 / 计数 / log
			}
		}
	}
}

func (r *WSReceiver[T]) closeConn() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		_ = r.conn.WriteControl(
			ws.CloseMessage,
			ws.FormatCloseMessage(ws.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		_ = r.conn.Close()
		r.conn = nil
		r.connected.Store(false)
	}
}

func (r *WSReceiver[T]) heartbeat() {
	ticker := time.NewTicker(r.heartBeat)
	defer func() {
		ticker.Stop()
		p := recover()
		if p != nil {
			fmt.Println("receive panic:", p)
		}
	}()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			if r.IsConnected() {
				r.mu.Lock()
				err := r.conn.WriteControl(ws.PingMessage, []byte{}, time.Now().Add(time.Second))
				r.mu.Unlock()
				if err != nil {
					r.Close()
					return
				}
			}
			break
		}
	}
}
