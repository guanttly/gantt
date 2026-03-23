package ws

import (
	"sync"
)

// IHub 连接管理中心接口
// 管理所有客户端连接，支持分组和广播
type IHub interface {
	// Register 注册客户端到指定分组
	// groupID 为空时只注册到全局列表
	Register(client *Client, groupID string)

	// Unregister 注销客户端
	Unregister(client *Client)

	// Broadcast 向指定分组的所有客户端广播消息
	// 返回成功发送的客户端数量
	Broadcast(groupID string, message []byte) int

	// BroadcastAll 向所有客户端广播消息
	// 返回成功发送的客户端数量
	BroadcastAll(message []byte) int

	// GetGroupClients 获取分组中的所有客户端
	GetGroupClients(groupID string) []*Client

	// GetAllClients 获取所有客户端
	GetAllClients() []*Client

	// GetClientGroup 获取客户端所属的分组 ID
	GetClientGroup(client *Client) (string, bool)

	// ClientCount 获取客户端总数
	ClientCount() int

	// GroupCount 获取分组总数
	GroupCount() int

	// GroupClientCount 获取指定分组的客户端数量
	GroupClientCount(groupID string) int
}

// Hub 连接管理中心
// 管理客户端连接，支持分组和广播
// 实现 IHub 接口
type Hub struct {
	mu sync.RWMutex

	// clients 所有连接的客户端
	clients map[*Client]struct{}

	// groups 客户端分组（如按 sessionID 分组）
	groups map[string]map[*Client]struct{}

	// clientGroups 客户端所属的组
	clientGroups map[*Client]string
}

// newHub 创建新的 Hub
func newHub() *Hub {
	return &Hub{
		clients:      make(map[*Client]struct{}),
		groups:       make(map[string]map[*Client]struct{}),
		clientGroups: make(map[*Client]string),
	}
}

// Register 注册客户端到指定分组
// groupID 为空时只注册到全局列表
func (h *Hub) Register(client *Client, groupID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 添加到全局客户端列表
	h.clients[client] = struct{}{}

	if groupID == "" {
		return
	}

	// 如果客户端已在其他组，先移除
	if oldGroupID, ok := h.clientGroups[client]; ok && oldGroupID != groupID {
		if oldGroup, exists := h.groups[oldGroupID]; exists {
			delete(oldGroup, client)
			if len(oldGroup) == 0 {
				delete(h.groups, oldGroupID)
			}
		}
	}

	// 添加到新组
	group := h.groups[groupID]
	if group == nil {
		group = make(map[*Client]struct{})
		h.groups[groupID] = group
	}
	group[client] = struct{}{}
	h.clientGroups[client] = groupID
}

// Unregister 注销客户端
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 从全局列表移除
	delete(h.clients, client)

	// 从分组移除
	if groupID, ok := h.clientGroups[client]; ok {
		if group, exists := h.groups[groupID]; exists {
			delete(group, client)
			if len(group) == 0 {
				delete(h.groups, groupID)
			}
		}
		delete(h.clientGroups, client)
	}
}

// Broadcast 向指定分组的所有客户端广播消息
func (h *Hub) Broadcast(groupID string, data []byte) int {
	h.mu.RLock()
	group := h.groups[groupID]
	if len(group) == 0 {
		h.mu.RUnlock()
		return 0
	}

	// 复制客户端列表避免长时间持锁
	clients := make([]*Client, 0, len(group))
	for client := range group {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	// 发送消息
	sentCount := 0
	for _, client := range clients {
		if client.Send(data) {
			sentCount++
		}
	}

	return sentCount
}

// BroadcastAll 向所有客户端广播消息
func (h *Hub) BroadcastAll(data []byte) int {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	sentCount := 0
	for _, client := range clients {
		if client.Send(data) {
			sentCount++
		}
	}

	return sentCount
}

// GetGroupClients 获取指定分组的所有客户端
func (h *Hub) GetGroupClients(groupID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	group := h.groups[groupID]
	clients := make([]*Client, 0, len(group))
	for client := range group {
		clients = append(clients, client)
	}

	return clients
}

// GetAllClients 获取所有客户端
func (h *Hub) GetAllClients() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}

	return clients
}

// GetClientGroup 获取客户端所属的分组 ID
func (h *Hub) GetClientGroup(client *Client) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	groupID, ok := h.clientGroups[client]
	return groupID, ok
}

// ClientCount 获取客户端总数
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GroupCount 获取分组总数
func (h *Hub) GroupCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.groups)
}

// GroupClientCount 获取指定分组的客户端数量
func (h *Hub) GroupClientCount(groupID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if group, ok := h.groups[groupID]; ok {
		return len(group)
	}
	return 0
}
