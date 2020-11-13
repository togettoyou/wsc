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
	ws := wsc.New("ws://127.0.0.1:9999/websocket/connect?key=123456")
	ws.OnConnected = func() {
		log.Println("OnConnected", ws.WebSocket.Url)
	}
	ws.OnConnectError = func(err error) {
		log.Println("OnConnectError", err)
	}
	ws.OnDisconnected = func(err error) {
		log.Println("OnDisconnected", err)
	}
	ws.OnClose = func(code int, text string) {
		log.Println("OnClose", code, text)
		done <- true
	}
	go ws.Connect()
	for {
		select {
		case <-done:
			return
		}
	}
}
```
