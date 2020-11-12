# golang websocket client
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/togettoyou/wsc)
[![GoDoc](https://godoc.org/github.com/togettoyou/wsc?status.svg)](https://godoc.org/github.com/togettoyou/wsc)

### Install

```
$ go get -v github.com/togettoyou/wsc
```
#### Simple example

``` go

func main() {
	done := make(chan bool)
	// 初始化
	ws := wsc.New("ws://127.0.0.1:9999/websocket/connect?key=123456")
	// 配置
	ws.Config.WriteWait = 12 * time.Second
	// -------------回调---------------
	ws.OnConnected = func() {
		fmt.Println("OnConnected", ws.WebSocket.Url)
	}
	ws.OnConnectError = func(err error) {
		fmt.Println("OnConnectError", err)
	}
	ws.OnDisconnected = func(err error) {
		fmt.Println("OnDisconnected", err)
	}
	ws.OnClose = func(code int, text string) {
		fmt.Println("OnClose", code, text)
		done <- true //退出程序
	}
	ws.OnPingReceived = func(appData string) {
		fmt.Println("OnPingReceived", appData)
	}
	ws.OnPongReceived = func(appData string) {
		fmt.Println("OnPongReceived", appData)
	}
	ws.OnBinaryMessageReceived = func(data []byte) {
		fmt.Println("OnBinaryMessageReceived", string(data))
	}
	ws.OnTextMessageReceived = func(message string) {
		fmt.Println("OnTextMessage", message)
	}
	ws.OnTextMessageSent = func(message string) {
		fmt.Println("OnTextMessageSent", message)
	}
	ws.OnBinaryMessageSent = func(data []byte) {
		fmt.Println("OnBinaryMessageSent", string(data))
	}
	// -------------回调---------------
	// 连接
	go ws.Connect()
	// 模拟
	go func() {
		t := time.NewTicker(2 * time.Second)
		quit := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-t.C:
				// 发送消息
				err := ws.SendBinaryMessage([]byte("hello wsc"))
				if err != nil {
					if err == wsc.CloseErr {
						fmt.Println("未连接")
					} else {
						fmt.Println(err.Error())
					}
				}
			case <-quit.C:
				fmt.Println("主动断开连接")
				ws.Close()
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
```
