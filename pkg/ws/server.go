package ws

import (
	"context"
	"net/http"

	"jusha/mcp/pkg/logging"

	"github.com/gorilla/websocket"
)

// MessageHandler 消息处理器函数类型
// 业务层实现此接口来处理 WebSocket 消息
type MessageHandler func(client *Client, data []byte) error

// IServer WebSocket 服务器接口
// 处理 HTTP 升级为 WebSocket 连接
type IWSServer interface {
	// HandleWS 处理 WebSocket 连接请求
	// 将 HTTP 连接升级为 WebSocket 连接
	HandleWS(w http.ResponseWriter, r *http.Request)

	// SetMessageHandler 设置消息处理器
	SetMessageHandler(handler MessageHandler)

	// GetHub 获取关联的 Hub
	GetHub() *Hub
}

// Upgrader WebSocket 升级器配置
var DefaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源（生产环境应限制）
	},
}

// Server WebSocket 服务器
// 实现 IServer 接口
type Server struct {
	hub      *Hub
	upgrader websocket.Upgrader
	logger   logging.ILogger
	handler  MessageHandler
}

// ServerOption 服务器配置选项
type ServerOption func(*Server)

// WithUpgrader 自定义升级器
func WithUpgrader(upgrader websocket.Upgrader) ServerOption {
	return func(s *Server) {
		s.upgrader = upgrader
	}
}

// WithMessageHandler 设置消息处理器
func WithMessageHandler(handler MessageHandler) ServerOption {
	return func(s *Server) {
		s.handler = handler
	}
}

// NewServer 创建 WebSocket 服务器
func newServer(hub *Hub, logger logging.ILogger, opts ...ServerOption) *Server {
	s := &Server{
		hub:      hub,
		upgrader: DefaultUpgrader,
		logger:   logger,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// HandleWS 处理 WebSocket 连接请求
func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("websocket upgrade failed", "error", err)
		http.Error(w, "upgrade failed", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	client := NewClient(ctx, conn, s.hub, s.logger)

	// 注册到 hub（不指定分组）
	s.hub.Register(client, "")

	// 启动写入循环
	go client.WritePump()

	// 启动读取循环（阻塞）
	if s.handler != nil {
		client.ReadPump(s.handler)
	} else {
		// 没有 handler，只维持连接
		client.ReadPump(func(c *Client, data []byte) error {
			return nil
		})
	}
}

// GetHub 获取 Hub
func (s *Server) GetHub() *Hub {
	return s.hub
}

// SetMessageHandler 设置消息处理器
func (s *Server) SetMessageHandler(handler MessageHandler) {
	s.handler = handler
}
