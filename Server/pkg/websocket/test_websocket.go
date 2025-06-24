// websocket/websocket_test.go
package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketCommunication(t *testing.T) {
	// 1. 创建并运行 Hub
	hub := NewHub()
	go hub.Run()

	// 2. 使用 httptest 创建一个测试服务器
	// 服务器的处理器就是我们的 WebSocket 服务
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	}))
	defer s.Close()

	// 3. 将 HTTP 服务器的 URL 转换为 WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

	// 4. 创建一个 WebSocket 客户端连接到测试服务器
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer ws.Close()

	// 5. 定义要广播的消息
	testMessage := []byte("hello, websocket")

	// 6. 通过 Hub 广播消息
	hub.Broadcast <- testMessage

	// 7. 从客户端读取消息并验证
	// 设置一个读取超时，防止测试永久阻塞
	ws.SetReadDeadline(time.Now().Add(time.Second))
	_, p, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if string(p) != string(testMessage) {
		t.Errorf("Received message %q, want %q", string(p), string(testMessage))
	}
}
