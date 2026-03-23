package session

import (
	"sort"
	"sync"

	"jusha/mcp/pkg/errors"
)

// Store 会话存储接口
// 提供会话的持久化和查询能力
type IStore interface {
	// Get 获取会话
	Get(id string) (*Session, error)

	// Set 保存会话
	Set(session *Session) error

	// Update 更新会话（支持 CAS 操作）
	// expectedVersion: 期望的版本号，用于乐观锁
	// mutate: 更新函数，接收当前会话并修改
	// 返回：是否成功、新版本号、错误
	Update(id string, expectedVersion int64, mutate func(*Session) error) (bool, int64, error)

	// Delete 删除会话
	Delete(id string) error

	// List 查询会话列表
	List(filter SessionFilter) ([]*Session, error)

	// Count 统计会话数量
	Count(filter SessionFilter) (int, error)
}

// SessionFilter 会话查询过滤器
type SessionFilter struct {
	OrgID     string       // 组织 ID
	UserID    string       // 用户 ID
	AgentType string       // Agent 类型
	State     SessionState // 状态
	Limit     int          // 限制数量
	Offset    int          // 偏移量
}

// InMemoryStore 内存存储实现
// 适用于开发和测试环境
type InMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewInMemoryStore 创建内存存储
func newInMemoryStore() IStore {
	return &InMemoryStore{
		sessions: make(map[string]*Session),
	}
}

// Get 获取会话
func (s *InMemoryStore) Get(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, nil // 不存在返回 nil，不是错误
	}

	// 返回副本避免并发修改
	return s.copySession(session), nil
}

// Set 保存会话
func (s *InMemoryStore) Set(session *Session) error {
	if session == nil {
		return errors.NewInvalidArgumentError("session is nil")
	}
	if session.ID == "" {
		return errors.NewInvalidArgumentError("session ID is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 保存副本
	s.sessions[session.ID] = s.copySession(session)
	return nil
}

// Update 更新会话（CAS）
func (s *InMemoryStore) Update(id string, expectedVersion int64, mutate func(*Session) error) (bool, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.sessions[id]
	if !ok {
		return false, 0, errors.NewNotFoundError("session not found", nil)
	}

	// 版本检查（乐观锁）
	if expectedVersion > 0 && current.Version != expectedVersion {
		return false, current.Version, nil
	}

	// 复制一份用于修改
	updated := s.copySession(current)

	// 执行修改
	if err := mutate(updated); err != nil {
		return false, current.Version, err
	}

	// 更新版本号
	updated.Version++
	// mutate 应该已经设置了 UpdatedAt

	// 保存
	s.sessions[id] = updated

	return true, updated.Version, nil
}

// Delete 删除会话
func (s *InMemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	return nil
}

// List 查询会话列表
func (s *InMemoryStore) List(filter SessionFilter) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Session

	for _, session := range s.sessions {
		if s.matchFilter(session, filter) {
			result = append(result, s.copySession(session))
		}
	}

	// 按 UpdatedAt 降序排序（最新的在前面）
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	// 应用分页
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	} else if filter.Offset >= len(result) {
		result = []*Session{}
	}

	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result, nil
}

// Count 统计会话数量
func (s *InMemoryStore) Count(filter SessionFilter) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, session := range s.sessions {
		if s.matchFilter(session, filter) {
			count++
		}
	}

	return count, nil
}

// matchFilter 检查会话是否匹配过滤器
func (s *InMemoryStore) matchFilter(session *Session, filter SessionFilter) bool {
	if filter.OrgID != "" && session.OrgID != filter.OrgID {
		return false
	}
	if filter.UserID != "" && session.UserID != filter.UserID {
		return false
	}
	if filter.AgentType != "" && session.AgentType != filter.AgentType {
		return false
	}
	if filter.State != "" && session.State != filter.State {
		return false
	}
	return true
}

// copySession 深拷贝会话（避免并发修改）
func (s *InMemoryStore) copySession(src *Session) *Session {
	if src == nil {
		return nil
	}

	dst := &Session{
		ID:        src.ID,
		OrgID:     src.OrgID,
		UserID:    src.UserID,
		AgentType: src.AgentType,
		State:     src.State,
		StateDesc: src.StateDesc,
		Error:     src.Error,
		Version:   src.Version,
		CreatedAt: src.CreatedAt,
		UpdatedAt: src.UpdatedAt,
		ExpireAt:  src.ExpireAt,
	}

	// 拷贝消息
	dst.Messages = make([]Message, len(src.Messages))
	copy(dst.Messages, src.Messages)

	// 拷贝工作流元数据
	if src.WorkflowMeta != nil {
		dst.WorkflowMeta = &WorkflowMeta{
			Workflow:            src.WorkflowMeta.Workflow,
			Version:             src.WorkflowMeta.Version,
			InstanceID:          src.WorkflowMeta.InstanceID,
			Description:         src.WorkflowMeta.Description,
			Phase:               src.WorkflowMeta.Phase,
			ActionsTransitionID: src.WorkflowMeta.ActionsTransitionID,
		}
		if src.WorkflowMeta.Actions != nil {
			dst.WorkflowMeta.Actions = make([]WorkflowAction, len(src.WorkflowMeta.Actions))
			copy(dst.WorkflowMeta.Actions, src.WorkflowMeta.Actions)
		}
		if src.WorkflowMeta.Extra != nil {
			dst.WorkflowMeta.Extra = make(map[string]any)
			for k, v := range src.WorkflowMeta.Extra {
				dst.WorkflowMeta.Extra[k] = v
			}
		}
	}

	// 拷贝 Data（浅拷贝，假设值是不可变的或业务层负责深拷贝）
	if src.Data != nil {
		dst.Data = make(map[string]any)
		for k, v := range src.Data {
			dst.Data[k] = v
		}
	}

	// 拷贝 Metadata
	if src.Metadata != nil {
		dst.Metadata = make(map[string]any)
		for k, v := range src.Metadata {
			dst.Metadata[k] = v
		}
	}

	return dst
}
