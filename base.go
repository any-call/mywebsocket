package mywebsocket

import (
	ws "github.com/gorilla/websocket"
)

type (
	Client interface {
		ID() string
		WriteMessage(data string) error
		WriteJson(data any) error
		WriteJsonByReadCb(data any, fn func(rData []byte) error) error
		IsConnect() bool
		Close()
	}

	ClientManager interface {
		Connect(conn *ws.Conn, id string) (Client, error)
		SendToClient(msg *Message)
	}

	ReadCBFun func(id string, data any)

	Message struct {
		Id     string //空代表发给所有的客户端
		IsJson bool
		Data   any
	}
)
