package main

import (
	"github.com/togettoyou/wsc"
	"log"
	"time"
)

func main() {
	done := make(chan bool)
	ws := wsc.New("ws://127.0.0.1:7777/ws")
	// 自定义配置，不使用默认配置
	ws.SetConfig(&wsc.Config{
		WriteWait:         10 * time.Second,
		MaxMessageSize:    2048,
		MinRecTime:        2 * time.Second,
		MaxRecTime:        60 * time.Second,
		RecFactor:         1.5,
		MessageBufferSize: 1024,
	})
	// 回调处理
	ws.OnConnected(func() {
		log.Println("OnConnected: ", ws.WebSocket.Url)
		// 连接成功后，测试每5秒发送消息
		go func() {
			t := time.NewTicker(5 * time.Second)
			for {
				select {
				case <-t.C:
					err := ws.SendTextMessage("hello")
					if err == wsc.CloseErr {
						return
					}
				}
			}
		}()
	})
	ws.OnConnectError(func(err error) {
		log.Println("OnConnectError: ", err.Error())
	})
	ws.OnDisconnected(func(err error) {
		log.Println("OnDisconnected: ", err.Error())
	})
	ws.OnClose(func(code int, text string) {
		log.Println("OnClose: ", code, text)
		done <- true
	})
	ws.OnTextMessageSent(func(message string) {
		log.Println("OnTextMessageSent: ", message)
	})
	ws.OnBinaryMessageSent(func(data []byte) {
		log.Println("OnBinaryMessageSent: ", string(data))
	})
	ws.OnSentError(func(err error) {
		log.Println("OnSentError: ", err.Error())
	})
	ws.OnPingReceived(func(appData string) {
		log.Println("OnPingReceived: ", appData)
	})
	ws.OnPongReceived(func(appData string) {
		log.Println("OnPongReceived: ", appData)
	})
	ws.OnTextMessageReceived(func(message string) {
		log.Println("OnTextMessageReceived: ", message)
	})
	ws.OnBinaryMessageReceived(func(data []byte) {
		log.Println("OnBinaryMessageReceived: ", string(data))
	})
	// 开始连接
	go ws.Connect()
	for {
		select {
		case <-done:
			return
		}
	}
}
