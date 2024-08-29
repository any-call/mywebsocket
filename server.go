package mywebsocket

import (
	ws "github.com/gorilla/websocket"
	"log"
	"net/http"
)

type server struct {
	addr     string
	upgrader ws.Upgrader
	connfun  ConnectFun
}

func NewServer(addr string, handleFn ConnectFun) Server {
	return &server{
		addr: addr,
		upgrader: ws.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		connfun: nil,
	}
}

func (self *server) Config(upgrader ws.Upgrader) {
	self.upgrader = upgrader
}

func (self *server) Start(route string) error {
	if route == "" {
		route = "/"
	}

	http.HandleFunc(route, self.handleConnections)
	return http.ListenAndServe(self.addr, nil)
}

func (self *server) StartTLS(route, certFile, keyFile string) error {
	if route == "" {
		route = "/"
	}

	http.HandleFunc(route, self.handleConnections)
	return http.ListenAndServeTLS(self.addr, certFile, keyFile, nil)
}

func (self *server) handleConnections(w http.ResponseWriter, r *http.Request) {
	// 升级 HTTP 连接为 WebSocket 连接
	conn, err := self.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	// 调用用户提供的连接处理函数
	if self.connfun == nil {
		_ = conn.Close()
		return
	}

	go self.connfun(conn)
}
