package mywebsocket

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	ws "github.com/gorilla/websocket"
)

func TestServer_Start(t *testing.T) {
	manager := NewClientManager(nil, nil, 100, handleReceiveMsg)
	ser := NewServer(":19080", func(conn *ws.Conn, r *http.Request) {
		fmt.Println("enter conn:", conn)
		if _, err := manager.Connect(conn, conn.RemoteAddr().String(), nil, ""); err != nil {
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

func handleReceiveMsg(id string, remoteIp string, data Envelope) {
	fmt.Println("received data is :", id, data)
}

func Test_Subscribe(t *testing.T) {
	//ws://64.176.53.2:19080
	//ws://127.0.0.1:19080
	header := http.Header{}
	header.Set("X-Client-ID", "myid")
	header.Set("X-Client-Subs", strings.Join([]string{"11", "22"}, ","))
	conn, _, err := ws.DefaultDialer.Dial("ws://64.176.53.2:19080", header)
	if err != nil {
		t.Error("conn err:", err)
		return
	}

	type BaseReq struct {
		Type   string   `json:"type"`
		Method string   `json:"method"`
		Params []string `json:"params"`
	}

	if err := conn.WriteJSON(BaseReq{
		Type:   "forex",
		Method: "ForexAggregatePerSec",
		Params: []string{"AED/AUD"},
	}); err != nil {
		t.Error("subscribe err:", err)
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

	ch := make(chan struct{}, 1)
	<-ch

	t.Log("run over")
}
