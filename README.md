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
	ws := wsc.New("ws://localhost:9999/websocket/connect")
	ws.OnConnected = func(ws wsc.WebSocket) {
		fmt.Println(ws.Url)
	}
	ws.OnClose = func(code int, text string, ws wsc.WebSocket) {
		done <- true
	}
	ws.Connect()
	for {
		select {
		case <-done:
			return
		}
	}
}
```
