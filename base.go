package mywebsocket

import (
	"net/http"
	"time"

	ws "github.com/gorilla/websocket"
)

type (
	Client interface {
		ID() string
		WriteMessage(data string) error
		WriteJson(data any) error
		WriteAndReadJson(data any, timeout time.Duration) ([]byte, error)
		IsConnect() bool
		Close()
	}

	ClientManager interface {
		TotalConn() int
		Connect(conn *ws.Conn, id string) (Client, error)
		SendToClient(msg *Message)
	}

	Server interface {
		Start(route string) error
		StartTLS(route, certFile, keyFile string) error
		Config(upgrader ws.Upgrader)
	}

	ConnectFun func(conn *ws.Conn, r *http.Request)
	ReadCBFun  func(id string, data any)

	Message struct {
		Id     string //空代表发给所有的客户端
		IsJson bool
		Data   any
	}
)
