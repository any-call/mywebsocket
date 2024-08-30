package mywebsocket

import (
	"fmt"
	ws "github.com/gorilla/websocket"
	"testing"
)

func TestServer_Start(t *testing.T) {
	manager := NewClientManager(nil, nil, handleReceiveMsg)
	ser := NewServer(":19050", func(conn *ws.Conn) {
		fmt.Println("enter conn:", conn)
		if _, err := manager.Connect(conn, conn.RemoteAddr().String()); err != nil {
			fmt.Println("manager.connect err:", err)
		} else {
			fmt.Println("manager total:", manager.TotalConn())
		}
	}, false)

	if err := ser.Start(""); err != nil {
		t.Error("start err:", err)
		return
	}

	t.Log("run ok")
}

func handleReceiveMsg(id string, data any) {
	fmt.Println("received data is :", id, data)
}
