// websocket/hub.go
package websocket

// Hub 维护所有活跃的客户端，并向它们广播消息。
type Hub struct {
	// 已注册的客户端。用 map 的 key 来存客户端指针，bool 值没有实际意义，只是为了利用 map 的快速查找。
	clients map[*Client]bool

	// 从客户端传入的消息将通过此 channel 广播。
	Broadcast chan []byte

	// 新客户端的注册请求。
	register chan *Client

	// 客户端的注销请求。
	unregister chan *Client
}

// NewHub 创建一个新的 Hub 实例。
func NewHub() *Hub {
	return &Hub{
		// 【核心修改】在这里为 Broadcast channel 添加缓冲区，大小为 256
		// 这允许gRPC服务在不阻塞的情况下，快速地向Hub发送最多256条日志消息。
		Broadcast: make(chan []byte, 256),

		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run 启动 Hub 的核心逻辑。它必须在一个单独的 goroutine 中运行。
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			// 当一个新客户端注册时，将其添加到 clients map 中。
			h.clients[client] = true
		case client := <-h.unregister:
			// 当一个客户端注销时，检查它是否存在，如果存在则删除。
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.Broadcast:
			// 当收到广播消息时，将其发送给所有已注册的客户端。
			for client := range h.clients {
				select {
				case client.send <- message:
					// 成功发送消息。
				default:
					// 如果客户端的发送 channel 已满，则认为该客户端已断开或处理缓慢。
					// 关闭并删除该客户端。
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
