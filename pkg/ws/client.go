// Package ws 提供通用的 WebSocket 连接管理
// 可独立使用，也可与其他模块集成
package ws

import (
	"context"
	"sync"
	"time"

	"jusha/mcp/pkg/logging"

	"github.com/gorilla/websocket"
)

// IClient WebSocket 客户端接口
// 封装单个 WebSocket 连接的生命周期管理
type IClient interface {
	// ID 获取客户端唯一标识
	ID() string

	// SetID 设置客户端唯一标识
	SetID(id string)

	// Send 发送消息（非阻塞）
	// 如果发送缓冲区满，消息将被丢弃
	Send(data []byte) bool

	// Close 关闭连接
	Close()

	// SetMetadata 设置客户端元数据
	SetMetadata(key string, value any)

	// GetMetadata 获取客户端元数据
	GetMetadata(key string) (any, bool)

	// GetAllMetadata 获取所有元数据
	GetAllMetadata() map[string]any

	// Context 获取客户端上下文
	Context() context.Context

	// WritePump 启动写消息循环（内部使用）
	WritePump()

	// ReadPump 启动读消息循环（内部使用）
	ReadPump(handler MessageHandler)
}

// Client WebSocket 客户端连接
type Client struct {
	ctx    context.Context
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	logger logging.ILogger
	id     string // 客户端唯一标识

	// 元数据（业务层可扩展）
	metadata map[string]any
	mu       sync.RWMutex

	// 关闭标志
	closed bool
}

// NewClient 创建新的客户端连接
func NewClient(ctx context.Context, conn *websocket.Conn, hub *Hub, logger logging.ILogger) *Client {
	return &Client{
		ctx:      ctx,
		conn:     conn,
		send:     make(chan []byte, 256),
		hub:      hub,
		logger:   logger,
		metadata: make(map[string]any),
	}
}

// ID 获取客户端 ID
func (c *Client) ID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id
}

// SetID 设置客户端 ID
func (c *Client) SetID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.id = id
}

// Context 获取上下文
func (c *Client) Context() context.Context {
	return c.ctx
}

// Send 发送消息到客户端
// 非阻塞，如果队列满则丢弃
func (c *Client) Send(data []byte) bool {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return false
	}
	c.mu.RUnlock()

	select {
	case c.send <- data:
		return true
	default:
		if c.logger != nil {
			c.logger.Warn("client send queue full, message dropped", "clientID", c.id)
		}
		return false
	}
}

// Close 关闭客户端连接
func (c *Client) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	if c.hub != nil {
		c.hub.Unregister(c)
	}
	close(c.send)
	_ = c.conn.Close()
}

// WritePump 消息写入循环
// 从 send 通道读取消息并写入 WebSocket
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道关闭
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				if c.logger != nil {
					c.logger.Error("websocket write error", "error", err, "clientID", c.id)
				}
				return
			}

		case <-ticker.C:
			// 发送心跳
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump 消息读取循环
// 从 WebSocket 读取消息并通过 handler 处理
func (c *Client) ReadPump(handler MessageHandler) {
	defer c.Close()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				if c.logger != nil {
					c.logger.Error("websocket read error", "error", err, "clientID", c.id)
				}
			}
			break
		}

		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// 调用业务处理函数
		if handler != nil {
			if err := handler(c, message); err != nil {
				if c.logger != nil {
					c.logger.Error("message handler error", "error", err, "clientID", c.id)
				}
			}
		}
	}
}

// SetMetadata 设置元数据
func (c *Client) SetMetadata(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metadata[key] = value
}

// GetMetadata 获取元数据
func (c *Client) GetMetadata(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.metadata[key]
	return val, ok
}

// GetAllMetadata 获取所有元数据
func (c *Client) GetAllMetadata() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]any, len(c.metadata))
	for k, v := range c.metadata {
		result[k] = v
	}
	return result
}
