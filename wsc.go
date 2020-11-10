package wsc

import (
	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type WebSocket struct {
	Url    string
	Conn   *websocket.Conn
	Dialer *websocket.Dialer

	RequestHeader http.Header
	HttpResponse  *http.Response

	// 连接成功回调
	OnConnected func(ws WebSocket)
	// 连接异常回调，在准备进行连接的过程中发生异常时触发
	OnConnectError func(err error, ws WebSocket)
	// 连接断开回调，网络异常，服务端掉线等情况时触发
	OnDisconnected func(err error, ws WebSocket)
	// 连接关闭回调，服务端发起关闭信号或客户端主动关闭时触发
	OnClose func(code int, text string, ws WebSocket)

	// 接受到Ping消息回调
	OnPingReceived func(appData string, ws WebSocket)
	// 接受到Pong消息回调
	OnPongReceived func(appData string, ws WebSocket)
	// 接受到Text消息回调
	OnTextMessage func(message string, ws WebSocket)
	// 接受到Binary消息回调
	OnBinaryMessage func(data []byte, ws WebSocket)

	// 最小重连时间间隔
	MinRecTime time.Duration
	// 最大重连时间间隔
	MaxRecTime time.Duration
	// 每次重连失败继续重连的时间间隔递增的乘数因子，递增到最大重连时间间隔为止
	RecFactor float64

	// 是否已连接
	IsConnected bool
	// 发送消息锁
	sendMu *sync.Mutex
	// 接受消息锁
	receiveMu *sync.Mutex
}

// 创建一个新的WebSocket客户端
func New(url string) WebSocket {
	return WebSocket{
		Url:           url,
		Dialer:        websocket.DefaultDialer,
		RequestHeader: http.Header{},
		MinRecTime:    2 * time.Second,
		MaxRecTime:    60 * time.Second,
		RecFactor:     1.5,
		sendMu:        &sync.Mutex{},
		receiveMu:     &sync.Mutex{},
	}
}

// 连接
func (ws *WebSocket) Connect() {
	b := &backoff.Backoff{
		Min:    ws.MinRecTime,
		Max:    ws.MaxRecTime,
		Factor: ws.RecFactor,
		Jitter: true,
	}
	rand.Seed(time.Now().UTC().UnixNano())
	for {
		var err error
		nextRec := b.Duration()
		ws.Conn, ws.HttpResponse, err = ws.Dialer.Dial(ws.Url, ws.RequestHeader)
		if err != nil {
			ws.IsConnected = false
			if ws.OnConnectError != nil {
				ws.OnConnectError(err, *ws)
			}
			// 重试
			time.Sleep(nextRec)
			continue
		}
		ws.IsConnected = true
		if ws.OnConnected != nil {
			ws.OnConnected(*ws)
		}
		defaultCloseHandler := ws.Conn.CloseHandler()
		ws.Conn.SetCloseHandler(func(code int, text string) error {
			result := defaultCloseHandler(code, text)
			ws.IsConnected = false
			if ws.OnClose != nil {
				ws.OnClose(code, text, *ws)
			}
			return result
		})
		defaultPingHandler := ws.Conn.PingHandler()
		ws.Conn.SetPingHandler(func(appData string) error {
			if ws.OnPingReceived != nil {
				ws.OnPingReceived(appData, *ws)
			}
			return defaultPingHandler(appData)
		})
		defaultPongHandler := ws.Conn.PongHandler()
		ws.Conn.SetPongHandler(func(appData string) error {
			if ws.OnPongReceived != nil {
				ws.OnPongReceived(appData, *ws)
			}
			return defaultPongHandler(appData)
		})
		go func() {
			for {
				ws.receiveMu.Lock()
				messageType, message, err := ws.Conn.ReadMessage()
				ws.receiveMu.Unlock()
				if err != nil {
					ws.IsConnected = false
					if ws.OnDisconnected != nil {
						ws.OnDisconnected(err, *ws)
					}
					ws.closeAndRecConn()
					return
				}
				switch messageType {
				case websocket.TextMessage:
					if ws.OnTextMessage != nil {
						ws.OnTextMessage(string(message), *ws)
					}
				case websocket.BinaryMessage:
					if ws.OnBinaryMessage != nil {
						ws.OnBinaryMessage(message, *ws)
					}
				}
			}
		}()
		return
	}
}

func (ws *WebSocket) SendTextMessage(message string) {
	err := ws.send(websocket.TextMessage, []byte(message))
	if err != nil {
	}
}

func (ws *WebSocket) SendBinaryMessage(data []byte) {
	err := ws.send(websocket.BinaryMessage, data)
	if err != nil {
	}
}

func (ws *WebSocket) send(messageType int, data []byte) error {
	var err error
	ws.sendMu.Lock()
	if ws.Conn != nil && ws.IsConnected {
		err = ws.Conn.WriteMessage(messageType, data)
	}
	ws.sendMu.Unlock()
	return err
}

// 断线重连
func (ws *WebSocket) closeAndRecConn() {
	if ws.Conn != nil {
		ws.Conn.Close()
	}
	go func() {
		ws.Connect()
	}()
}

// 主动关闭连接
func (ws *WebSocket) Close() {
	if ws.Conn != nil {
		ws.send(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		ws.Conn.Close()
		ws.IsConnected = false
		if ws.OnClose != nil {
			ws.OnClose(websocket.CloseNormalClosure, "", *ws)
		}
	}
}

func (ws *WebSocket) CloseWithMsg(msg string) {
	if ws.Conn != nil {
		ws.send(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, msg))
		ws.Conn.Close()
		ws.IsConnected = false
		if ws.OnClose != nil {
			ws.OnClose(websocket.CloseNormalClosure, msg, *ws)
		}
	}
}
