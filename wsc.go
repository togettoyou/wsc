package wsc

import (
	"errors"
	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Wsc struct {
	// 配置信息
	Config *Config
	// 底层WebSocket
	WebSocket *WebSocket
	// 连接成功回调
	OnConnected func()
	// 连接异常回调，在准备进行连接的过程中发生异常时触发
	OnConnectError func(err error)
	// 连接断开回调，网络异常，服务端掉线等情况时触发
	OnDisconnected func(err error)
	// 连接关闭回调，服务端发起关闭信号或客户端主动关闭时触发
	OnClose func(code int, text string)

	// 发送Text消息成功回调
	OnTextMessageSent func(message string)
	// 发送Binary消息成功回调
	OnBinaryMessageSent func(data []byte)

	// 接受到Ping消息回调
	OnPingReceived func(appData string)
	// 接受到Pong消息回调
	OnPongReceived func(appData string)
	// 接受到Text消息回调
	OnTextMessageReceived func(message string)
	// 接受到Binary消息回调
	OnBinaryMessageReceived func(data []byte)
}

type Config struct {
	// 写超时
	WriteWait time.Duration
	// 支持接受的消息最大长度，默认512字节
	MaxMessageSize int64
	// 最小重连时间间隔
	MinRecTime time.Duration
	// 最大重连时间间隔
	MaxRecTime time.Duration
	// 每次重连失败继续重连的时间间隔递增的乘数因子，递增到最大重连时间间隔为止
	RecFactor float64
}

type WebSocket struct {
	// 连接url
	Url           string
	Conn          *websocket.Conn
	Dialer        *websocket.Dialer
	RequestHeader http.Header
	HttpResponse  *http.Response
	// 是否已连接
	isConnected bool
	// 加锁避免重复关闭管道
	connMu *sync.RWMutex
	// 发送消息锁
	sendMu *sync.Mutex
}

// 创建一个Wsc客户端
func New(url string) *Wsc {
	return &Wsc{
		Config: &Config{
			WriteWait:      10 * time.Second,
			MaxMessageSize: 512,
			MinRecTime:     2 * time.Second,
			MaxRecTime:     60 * time.Second,
			RecFactor:      1.5,
		},
		WebSocket: &WebSocket{
			Url:           url,
			Dialer:        websocket.DefaultDialer,
			RequestHeader: http.Header{},
			isConnected:   false,
			connMu:        &sync.RWMutex{},
			sendMu:        &sync.Mutex{},
		},
	}
}

// 返回关闭状态
func (wsc *Wsc) Closed() bool {
	wsc.WebSocket.connMu.RLock()
	defer wsc.WebSocket.connMu.RUnlock()
	return !wsc.WebSocket.isConnected
}

// 发起连接
func (wsc *Wsc) Connect() {
	b := &backoff.Backoff{
		Min:    wsc.Config.MinRecTime,
		Max:    wsc.Config.MaxRecTime,
		Factor: wsc.Config.RecFactor,
		Jitter: true,
	}
	rand.Seed(time.Now().UTC().UnixNano())
	for {
		var err error
		nextRec := b.Duration()
		wsc.WebSocket.Conn, wsc.WebSocket.HttpResponse, err =
			wsc.WebSocket.Dialer.Dial(wsc.WebSocket.Url, wsc.WebSocket.RequestHeader)
		if err != nil {
			if wsc.OnConnectError != nil {
				wsc.OnConnectError(err)
			}
			// 重试
			time.Sleep(nextRec)
			continue
		}
		// 变更连接状态
		wsc.WebSocket.connMu.Lock()
		wsc.WebSocket.isConnected = true
		wsc.WebSocket.connMu.Unlock()
		// 连接成功回调
		if wsc.OnConnected != nil {
			wsc.OnConnected()
		}
		// 设置支持接受的消息最大长度
		wsc.WebSocket.Conn.SetReadLimit(wsc.Config.MaxMessageSize)
		// 超时时间
		wsc.WebSocket.Conn.SetWriteDeadline(time.Now().Add(wsc.Config.WriteWait))
		// 连接关闭回调
		defaultCloseHandler := wsc.WebSocket.Conn.CloseHandler()
		wsc.WebSocket.Conn.SetCloseHandler(func(code int, text string) error {
			result := defaultCloseHandler(code, text)
			wsc.Close()
			if wsc.OnClose != nil {
				wsc.OnClose(code, text)
			}
			return result
		})
		// 收到ping回调
		defaultPingHandler := wsc.WebSocket.Conn.PingHandler()
		wsc.WebSocket.Conn.SetPingHandler(func(appData string) error {
			if wsc.OnPingReceived != nil {
				wsc.OnPingReceived(appData)
			}
			return defaultPingHandler(appData)
		})
		// 收到pong回调
		defaultPongHandler := wsc.WebSocket.Conn.PongHandler()
		wsc.WebSocket.Conn.SetPongHandler(func(appData string) error {
			if wsc.OnPongReceived != nil {
				wsc.OnPongReceived(appData)
			}
			return defaultPongHandler(appData)
		})
		// 开启协程读
		go func() {
			for {
				messageType, message, err := wsc.WebSocket.Conn.ReadMessage()
				if err != nil {
					// 异常断线重连
					if wsc.OnDisconnected != nil {
						wsc.OnDisconnected(err)
					}
					wsc.closeAndRecConn()
					return
				}
				switch messageType {
				// 收到TextMessage回调
				case websocket.TextMessage:
					if wsc.OnTextMessageReceived != nil {
						wsc.OnTextMessageReceived(string(message))
					}
				// 收到BinaryMessage回调
				case websocket.BinaryMessage:
					if wsc.OnBinaryMessageReceived != nil {
						wsc.OnBinaryMessageReceived(message)
					}
				}
			}
		}()
		return
	}
}

var closeErr = errors.New("connection closed")

// 发送TextMessage消息
func (wsc *Wsc) SendTextMessage(message string) error {
	if wsc.Closed() {
		return closeErr
	}
	err := wsc.send(websocket.TextMessage, []byte(message))
	if err != nil {
		return err
	}
	if wsc.OnTextMessageSent != nil {
		wsc.OnTextMessageSent(message)
	}
	return nil
}

// 发送BinaryMessage消息
func (wsc *Wsc) SendBinaryMessage(data []byte) error {
	if wsc.Closed() {
		return closeErr
	}
	err := wsc.send(websocket.BinaryMessage, data)
	if err != nil {
		return err
	}
	if wsc.OnBinaryMessageSent != nil {
		wsc.OnBinaryMessageSent(data)
	}
	return nil
}

// 发送消息到连接端
func (wsc *Wsc) send(messageType int, data []byte) error {
	var err error
	wsc.WebSocket.sendMu.Lock()
	err = wsc.WebSocket.Conn.WriteMessage(messageType, data)
	wsc.WebSocket.sendMu.Unlock()
	return err
}

// 断线重连
func (wsc *Wsc) closeAndRecConn() {
	if wsc.Closed() {
		return
	}
	wsc.WebSocket.connMu.Lock()
	defer wsc.WebSocket.connMu.Unlock()
	wsc.WebSocket.isConnected = false
	wsc.WebSocket.Conn.Close()
	go func() {
		wsc.Connect()
	}()
}

// 主动关闭连接
func (wsc *Wsc) Close() {
	wsc.CloseWithMsg("")
}

// 主动关闭连接，附带消息
func (wsc *Wsc) CloseWithMsg(msg string) {
	if wsc.Closed() {
		return
	}
	wsc.WebSocket.connMu.Lock()
	defer wsc.WebSocket.connMu.Unlock()
	wsc.send(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, msg))
	wsc.WebSocket.isConnected = false
	wsc.WebSocket.Conn.Close()
	if wsc.OnClose != nil {
		wsc.OnClose(websocket.CloseNormalClosure, msg)
	}
}
