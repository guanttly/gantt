// Package websocket 提供 WebSocket 连接管理与消息广播。
package websocket

import (
	"encoding/json"
	"sync"
)

// Broadcaster 消息广播接口，供业务层（如排班 Pipeline）推送事件。
type Broadcaster interface {
	// BroadcastToGroup 向指定分组的所有连接推送 JSON 消息。
	BroadcastToGroup(groupID string, payload any) error

	// BroadcastAll 向所有连接推送 JSON 消息。
	BroadcastAll(payload any) error
}

// Conn 代表一条 WebSocket 连接。
type Conn struct {
	ID      string
	GroupID string
	Send    chan []byte
}

// Hub 连接管理中心，管理客户端分组与广播。
type Hub struct {
	mu     sync.RWMutex
	groups map[string]map[*Conn]struct{}
	conns  map[*Conn]struct{}
}

// NewHub 创建 Hub 实例。
func NewHub() *Hub {
	return &Hub{
		groups: make(map[string]map[*Conn]struct{}),
		conns:  make(map[*Conn]struct{}),
	}
}

// Register 注册一条连接到指定分组。
func (h *Hub) Register(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.conns[c] = struct{}{}
	if c.GroupID != "" {
		if h.groups[c.GroupID] == nil {
			h.groups[c.GroupID] = make(map[*Conn]struct{})
		}
		h.groups[c.GroupID][c] = struct{}{}
	}
}

// Unregister 注销连接。
func (h *Hub) Unregister(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if c.GroupID != "" {
		if g, ok := h.groups[c.GroupID]; ok {
			delete(g, c)
			if len(g) == 0 {
				delete(h.groups, c.GroupID)
			}
		}
	}
	delete(h.conns, c)
	close(c.Send)
}

// BroadcastToGroup 向指定分组发送消息。
func (h *Hub) BroadcastToGroup(groupID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.groups[groupID] {
		select {
		case c.Send <- data:
		default:
			// 跳过阻塞的连接
		}
	}
	return nil
}

// BroadcastAll 向所有连接发送消息。
func (h *Hub) BroadcastAll(payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.conns {
		select {
		case c.Send <- data:
		default:
		}
	}
	return nil
}

// GroupCount 返回指定分组的连接数。
func (h *Hub) GroupCount(groupID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.groups[groupID])
}

// ConnCount 返回总连接数。
func (h *Hub) ConnCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

// 确保 Hub 实现 Broadcaster 接口。
var _ Broadcaster = (*Hub)(nil)
