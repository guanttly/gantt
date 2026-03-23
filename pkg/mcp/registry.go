package mcp

import (
	"fmt"
	"jusha/mcp/pkg/mcp/model"
	"sync"
)

// MemoryToolRegistry 内存实现的工具注册表
type MemoryToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ITool
}

func NewMemoryToolRegistry() *MemoryToolRegistry {
	return &MemoryToolRegistry{
		tools: make(map[string]ITool),
	}
}

func (r *MemoryToolRegistry) RegisterTool(tool ITool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	r.tools[name] = tool
	return nil
}

func (r *MemoryToolRegistry) UnregisterTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(r.tools, name)
	return nil
}

func (r *MemoryToolRegistry) GetTool(name string) (ITool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if name == "" {
		return nil, fmt.Errorf("tool name cannot be empty")
	}

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return tool, nil
}

func (r *MemoryToolRegistry) ListTools() []model.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]model.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, model.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		})
	}

	return tools
}

func (r *MemoryToolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]ITool)
}

func (r *MemoryToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

func (r *MemoryToolRegistry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tools[name]
	return exists
}
