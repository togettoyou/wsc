package wsc

import (
	"testing"
	"time"
)

func TestWSC(t *testing.T) {
	done := make(chan bool)
	wsc := New("ws://localhost:9999/websocket/connect?key=123456")
	wsc.OnConnected = func(ws WebSocket) {
		t.Log(time.Now().Format("2006-01-02 15:04:05"), "OnConnected", ws.Url)
	}
	wsc.OnConnectError = func(err error, ws WebSocket) {
		t.Error(time.Now().Format("2006-01-02 15:04:05"), "OnConnectError", err)
	}
	wsc.OnDisconnected = func(err error, ws WebSocket) {
		t.Error(time.Now().Format("2006-01-02 15:04:05"), "OnDisconnected", err)
	}
	wsc.OnClose = func(code int, text string, ws WebSocket) {
		t.Log(time.Now().Format("2006-01-02 15:04:05"), "OnClose", code, text)
		done <- true //退出程序
	}
	wsc.OnPingReceived = func(appData string, ws WebSocket) {
		t.Log(time.Now().Format("2006-01-02 15:04:05"), "OnPingReceived", appData)
	}
	wsc.OnPongReceived = func(appData string, ws WebSocket) {
		t.Log(time.Now().Format("2006-01-02 15:04:05"), "OnPongReceived", appData)
	}
	wsc.OnBinaryMessage = func(data []byte, ws WebSocket) {
		t.Log(time.Now().Format("2006-01-02 15:04:05"), "OnBinaryMessage", string(data))
	}
	wsc.OnTextMessage = func(message string, ws WebSocket) {
		t.Log(time.Now().Format("2006-01-02 15:04:05"), "OnTextMessage", message)
	}
	wsc.Connect()
	go func() {
		t1 := time.Tick(3 * time.Second)
		t2 := time.Tick(30 * time.Second)
		for {
			select {
			case <-t1:
				wsc.SendBinaryMessage([]byte("hello wsc"))
			case <-t2:
				wsc.Close()
			}
		}
	}()
	for {
		select {
		case <-done:
			return
		}
	}
}
