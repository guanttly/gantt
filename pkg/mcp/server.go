package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"jusha/mcp/pkg/errors"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp/model"
)

// MCPServer MCP服务器接口
type MCPServer interface {
	Initialize(ctx context.Context, req *model.InitializeRequest) (*model.InitializeResult, error)
	ListTools(ctx context.Context, req *model.ListToolsRequest) (*model.ListToolsResult, error)
	CallTool(ctx context.Context, req *model.CallToolRequest) (*model.CallToolResult, error)
	RegisterTool(tool ITool) error
	UnregisterTool(name string) error
	HandleMessage(ctx context.Context, data []byte) ([]byte, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// DefaultMCPServer 默认MCP服务器实现
type DefaultMCPServer struct {
	serverInfo   model.ServerInfo
	capabilities model.ServerCapabilities
	registry     IToolRegistry
	logger       logging.ILogger
	mu           sync.RWMutex
}

func NewMCPServer(name, version string, logger logging.ILogger) MCPServer {
	return &DefaultMCPServer{
		serverInfo: model.ServerInfo{
			Name:    name,
			Version: version,
		},
		capabilities: model.ServerCapabilities{
			Tools: &model.ToolsCapability{},
		},
		registry: NewMemoryToolRegistry(),
		logger:   logger,
	}
}

func (s *DefaultMCPServer) ListTools(ctx context.Context, req *model.ListToolsRequest) (*model.ListToolsResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := s.registry.ListTools()
	s.logger.Debug("Listing tools", "count", len(tools))

	return &model.ListToolsResult{
		Tools: tools,
	}, nil
}

func (s *DefaultMCPServer) CallTool(ctx context.Context, req *model.CallToolRequest) (*model.CallToolResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tool, err := s.registry.GetTool(req.Name)
	if err != nil {
		s.logger.Error("Tool not found", "tool", req.Name, "error", err)
		return nil, NewToolError(errors.MCP_TOOL_NOT_FOUND, fmt.Sprintf("tool not found: %s", req.Name))
	}

	s.logger.Info("Executing tool", "tool", req.Name, "arguments", req.Arguments)

	result, err := tool.Execute(ctx, req.Arguments)
	if err != nil {
		s.logger.Error("Tool execution failed", "tool", req.Name, "error", err)
		// 如果已经是 ToolError，直接返回
		if _, ok := IsToolError(err); ok {
			return nil, err
		}
		// 否则包装为 ToolError
		return nil, WrapToolError(errors.MCP_TOOL_EXEC_ERROR, fmt.Sprintf("tool execution failed: %s", req.Name), err)
	}

	s.logger.Debug("Tool executed successfully", "tool", req.Name)
	return result, nil
}

func (s *DefaultMCPServer) Initialize(ctx context.Context, req *model.InitializeRequest) (*model.InitializeResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &model.InitializeResult{
		ProtocolVersion: model.MCPVersion,
		Capabilities:    s.capabilities,
		ServerInfo:      s.serverInfo,
	}, nil
}

func (s *DefaultMCPServer) RegisterTool(tool ITool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.registry.RegisterTool(tool)
	if err != nil {
		s.logger.Error("Failed to register tool", "tool", tool.Name(), "error", err)
		return err
	}

	s.logger.Info("Tool registered", "tool", tool.Name())
	return nil
}

func (s *DefaultMCPServer) UnregisterTool(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.registry.UnregisterTool(name)
	if err != nil {
		s.logger.Error("Failed to unregister tool", "tool", name, "error", err)
		return err
	}

	s.logger.Info("Tool unregistered", "tool", name)
	return nil
}

func (s *DefaultMCPServer) Start(ctx context.Context) error {
	s.logger.Info("MCP server starting", "name", s.serverInfo.Name, "version", s.serverInfo.Version)
	return nil
}

func (s *DefaultMCPServer) Stop(ctx context.Context) error {
	s.logger.Info("MCP server stopping")
	return nil
}

// HandleMessage 处理JSON-RPC消息
func (s *DefaultMCPServer) HandleMessage(ctx context.Context, data []byte) ([]byte, error) {
	msg, err := model.UnmarshalMCPMessage(data)
	if err != nil {
		return s.marshalErrorResponse(nil, int(errors.RPC_PARSE_ERROR), "Parse error", err.Error())
	}

	s.logger.Debug("Handling message", "method", msg.Method, "id", msg.ID)

	// 处理请求
	if msg.IsRequest() {
		return s.handleRequest(ctx, msg)
	}

	return nil, fmt.Errorf("unsupported message type")
}

func (s *DefaultMCPServer) handleRequest(ctx context.Context, msg *model.MCPMessage) ([]byte, error) {
	var result any
	var err error

	switch msg.Method {
	case "initialize":
		var req model.InitializeRequest
		if err := json.Unmarshal(mustMarshal(msg.Params), &req); err != nil {
			return s.marshalErrorResponse(msg.ID, int(errors.RPC_INVALID_PARAMS), "Invalid params", err.Error())
		}
		result, err = s.Initialize(ctx, &req)

	case "tools/list":
		var req model.ListToolsRequest
		result, err = s.ListTools(ctx, &req)

	case "tools/call":
		var req model.CallToolRequest
		if err := json.Unmarshal(mustMarshal(msg.Params), &req); err != nil {
			return s.marshalErrorResponse(msg.ID, int(errors.RPC_INVALID_PARAMS), "Invalid params", err.Error())
		}
		result, err = s.CallTool(ctx, &req)

	default:
		return s.marshalErrorResponse(msg.ID, int(errors.RPC_METHOD_NOT_FOUND), "Method not found", nil)
	}

	if err != nil {
		// 检查是否为 ToolError，提取业务错误码
		if toolErr, ok := IsToolError(err); ok {
			return s.marshalErrorResponse(msg.ID, int(toolErr.Code()), toolErr.Message(), toolErr.Data())
		}
		// 其他错误使用内部错误码
		return s.marshalErrorResponse(msg.ID, int(errors.RPC_INTERNAL_ERROR), "Internal error", err.Error())
	}

	response := &model.MCPMessage{
		JSONRPCMessage: model.JSONRPCMessage{
			JSONRpc: "2.0",
			ID:      msg.ID,
			Result:  result,
		},
	}

	return model.MarshalMCPMessage(response)
}

func (s *DefaultMCPServer) marshalErrorResponse(id any, code int, message string, data any) ([]byte, error) {
	response := &model.MCPMessage{
		JSONRPCMessage: model.JSONRPCMessage{
			JSONRpc: "2.0",
			ID:      id,
			Error: &model.RPCError{
				Code:    code,
				Message: message,
				Data:    data,
			},
		},
	}

	return model.MarshalMCPMessage(response)
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
