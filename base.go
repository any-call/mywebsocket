package mywebsocket

import (
	"net/http"
	"time"

	ws "github.com/gorilla/websocket"
)

type (
	Client interface {
		ID() string
		Subscriptions() []string
		WriteMessage(data string) error
		WriteJson(data any) error
		WriteAndReadJson(data any, timeout time.Duration) ([]byte, error)
		IsConnect() bool
		Close()
	}

	ClientManager interface {
		TotalConn() int
		Connect(conn *ws.Conn, id string, subScription []string) (Client, error)
		SendToClient(msg *Message)
		RangeConn(func(id string, c Client) bool)
	}

	Server interface {
		Start(route string) error
		StartTLS(route, certFile, keyFile string) error
		Config(upgrader ws.Upgrader)
	}

	ConnectFun func(conn *ws.Conn, r *http.Request)
	ReadCBFun  func(id string, data any)

	Message struct {
		// ===== 路由层 =====
		ID     string   // 精确 ID
		Topics []string // 根据订阅转发

		// ===== 数据层 =====
		IsJson bool
		Data   any
	}
)
