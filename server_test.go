package mywebsocket

import (
	"fmt"
	ws "github.com/gorilla/websocket"
	"testing"
)

func TestServer_Start(t *testing.T) {
	manager := NewClientManager(nil, nil, handleReceiveMsg)
	ser := NewServer(":19080", func(conn *ws.Conn) {
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

func Test_Subscribe(t *testing.T) {
	//ws://64.176.53.2:19080
	//ws://127.0.0.1:19080
	conn, _, err := ws.DefaultDialer.Dial("ws://64.176.53.2:19080", nil)
	if err != nil {
		t.Error("conn err:", err)
		return
	}

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				t.Error("read msg err:", err)
				break
			}
			t.Log("read msg:", string(msg))
		}
	}()

	type BaseReq struct {
		Method string `json:"method"`
		Params string `json:"params"`
	}

	if err := conn.WriteJSON(BaseReq{
		Method: "ForexAggregatePerSec",
		Params: "",
	}); err != nil {
		t.Error("subscribe err:", err)
		return
	}

	ch := make(chan struct{}, 1)
	<-ch

	t.Log("run over")
}
