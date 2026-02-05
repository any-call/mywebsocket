package mywebsocket

import (
	"encoding/json"
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
	ReadCBFun  func(id string, data Envelope)

	//收到消息中转结构定义
	Message struct {
		// ===== 路由层 =====
		ID     string   // 精确 ID
		Topics []string // 根据订阅转发

		// ===== 数据层 =====
		IsJson bool
		Data   any
	}

	//信封定义，各端发送的标准格式
	Envelope struct {
		Type  string `json:"type"` // log / metric / cmd / ack ...
		From  string `json:"from"` // 节点 ID（服务端可补）
		To    string `json:"to,omitempty"`
		Topic string `json:"topic,omitempty"` //扩展标识

		Data json.RawMessage `json:"data"` //真正的业务数据
	}
)
