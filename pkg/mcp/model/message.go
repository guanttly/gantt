package model

import (
	"encoding/json"
	"fmt"
	"jusha/mcp/pkg/errors"
)

// JSON-RPC 2.0 消息类型
type JSONRPCMessage struct {
	JSONRpc string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Method  string    `json:"method,omitempty"`
	Params  any       `json:"params,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// MCP协议消息类型
type MCPMessage struct {
	JSONRPCMessage
}

// 消息验证函数
func (m *MCPMessage) Validate() error {
	if m.JSONRpc != "2.0" {
		return errors.NewInvalidRequestError("invalid jsonrpc version")
	}

	if m.IsRequest() && m.Method == "" {
		return errors.NewInvalidRequestError("missing method for request")
	}

	if m.IsResponse() && m.ID == nil {
		return errors.NewInvalidRequestError("missing id for response")
	}

	return nil
}

func (m *MCPMessage) IsRequest() bool {
	return m.Method != "" && m.Result == nil && m.Error == nil
}

func (m *MCPMessage) IsResponse() bool {
	return m.Method == "" && (m.Result != nil || m.Error != nil)
}

func (m *MCPMessage) IsNotification() bool {
	return m.Method != "" && m.ID == nil
}

// JSON序列化辅助
func MarshalMCPMessage(msg *MCPMessage) ([]byte, error) {
	msg.JSONRpc = "2.0"
	return json.Marshal(msg)
}

func UnmarshalMCPMessage(data []byte) (*MCPMessage, error) {
	var msg MCPMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MCP message: %w", err)
	}

	if msg.JSONRpc != "2.0" {
		return nil, fmt.Errorf("invalid JSON-RPC version: %s", msg.JSONRpc)
	}

	return &msg, nil
}
