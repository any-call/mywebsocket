package mywebsocket

import (
	"github.com/gorilla/websocket"
	"net/http"
)

type (
	Client interface {
		ID() string
		WriteMessage(data string) error
		WriteJson(data any) error
		WriteJsonByReadCb(data any, fn func(rData []byte) error) error
		ReadChan() <-chan any
		IsConnect() bool
		Connect() error
		Close()
	}
)

// 升级器，用于将 HTTP 连接升级为 WebSocket 连接
func DefUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 默认允许所有连接
		},
	}
}
