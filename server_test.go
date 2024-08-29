package mywebsocket

import (
	"fmt"
	ws "github.com/gorilla/websocket"
	"testing"
)

func TestServer_Start(t *testing.T) {
	ser := NewServer(":19050", func(conn *ws.Conn) {
		fmt.Println("enter conn:", conn)
	}, false)

	if err := ser.Start("/ws"); err != nil {
		t.Error("start err:", err)
		return
	}

	t.Log("run ok")
}
