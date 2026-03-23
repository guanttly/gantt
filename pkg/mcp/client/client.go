package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp/model"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// MCPClient MCP客户端接口
type MCPClient interface {
	Initialize(ctx context.Context, clientInfo model.ClientInfo) error
	ListTools(ctx context.Context) ([]model.Tool, error)
	CallTool(ctx context.Context, name string, arguments map[string]any) (*model.CallToolResult, error)
	Close() error
}

// HTTPMCPClient HTTP传输的MCP客户端
type HTTPMCPClient struct {
	endpoint    string
	httpClient  *http.Client
	logger      logging.ILogger
	mu          sync.RWMutex
	initialized bool
	requestID   int64
}

func NewHTTPMCPClientWithTimeout(endpoint string, logger logging.ILogger, timeout time.Duration) *HTTPMCPClient {
	return &HTTPMCPClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

func (c *HTTPMCPClient) Initialize(ctx context.Context, clientInfo model.ClientInfo) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := &model.InitializeRequest{
		ProtocolVersion: model.MCPVersion,
		Capabilities: model.ClientCapabilities{
			Experimental: make(map[string]any),
		},
		ClientInfo: clientInfo,
	}

	var result model.InitializeResult
	if err := c.sendRequest(ctx, "initialize", req, &result); err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	c.initialized = true
	c.logger.Debug("MCP client initialized",
		"server", result.ServerInfo.Name,
		"version", result.ServerInfo.Version)

	return nil
}

func (c *HTTPMCPClient) ListTools(ctx context.Context) ([]model.Tool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	var result model.ListToolsResult
	if err := c.sendRequest(ctx, "tools/list", &model.ListToolsRequest{}, &result); err != nil {
		return nil, fmt.Errorf("list tools failed: %w", err)
	}

	return result.Tools, nil
}

func (c *HTTPMCPClient) CallTool(ctx context.Context, name string, arguments map[string]any) (*model.CallToolResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	// 如果arguments为空map，设置为nil以便JSON序列化时省略该字段
	var args map[string]any
	if len(arguments) > 0 {
		args = arguments
	}

	req := &model.CallToolRequest{
		Name:      name,
		Arguments: args,
	}

	var result model.CallToolResult
	if err := c.sendRequest(ctx, "tools/call", req, &result); err != nil {
		return nil, fmt.Errorf("call tool failed: %w", err)
	}

	return &result, nil
}

func (c *HTTPMCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initialized = false
	c.logger.Debug("MCP client closed")
	return nil
}

func (c *HTTPMCPClient) sendRequest(ctx context.Context, method string, params any, result any) error {
	requestID := atomic.AddInt64(&c.requestID, 1)

	request := &model.MCPMessage{
		JSONRPCMessage: model.JSONRPCMessage{
			JSONRpc: "2.0",
			ID:      requestID,
			Method:  method,
			Params:  params,
		},
	}

	requestData, err := model.MarshalMCPMessage(request)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	c.logger.Debug("Sending request", "method", method, "id", requestID)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(requestData))
	if err != nil {
		return fmt.Errorf("create http request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	response, err := model.UnmarshalMCPMessage(responseData)
	if err != nil {
		return fmt.Errorf("unmarshal response failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("RPC error %d: %s", response.Error.Code, response.Error.Message)
	}

	if response.Result == nil {
		return fmt.Errorf("empty result")
	}

	// 将结果反序列化到目标结构
	resultData, err := json.Marshal(response.Result)
	if err != nil {
		return fmt.Errorf("marshal result failed: %w", err)
	}

	if err := json.Unmarshal(resultData, result); err != nil {
		return fmt.Errorf("unmarshal result failed: %w", err)
	}

	c.logger.Debug("Request completed", "method", method, "id", requestID)
	return nil
}

// MCPClientPool 客户端连接池
type MCPClientPool struct {
	endpoint          string
	clientInfo        model.ClientInfo
	logger            logging.ILogger
	mu                sync.Mutex
	clients           []MCPClient
	httpClientTimeout time.Duration
	maxClients        int
	initialized       bool
}

func NewMCPClientPool(cfg config.MCPConfig, endpoint string, clientInfo model.ClientInfo, maxClients int, logger logging.ILogger) *MCPClientPool {
	// 根据配置创建MCP管理器
	clientTimeout := time.Duration(cfg.ClientTimeout) * time.Second

	// 如果配置为0，使用默认值
	if clientTimeout == 0 {
		clientTimeout = 5 * time.Minute // 默认5分钟
	}
	return &MCPClientPool{
		endpoint:          endpoint,
		clientInfo:        clientInfo,
		httpClientTimeout: clientTimeout,
		logger:            logger,
		maxClients:        maxClients,
	}
}

func (p *MCPClientPool) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	// 创建初始连接
	for i := 0; i < p.maxClients; i++ {
		client := NewHTTPMCPClientWithTimeout(p.endpoint, p.logger, p.httpClientTimeout)
		if err := client.Initialize(ctx, p.clientInfo); err != nil {
			// 清理已创建的客户端
			for _, c := range p.clients {
				c.Close()
			}
			return fmt.Errorf("initialize client %d failed: %w", i, err)
		}
		p.clients = append(p.clients, client)
	}

	p.initialized = true
	p.logger.Info("Client pool initialized", "endpoint", p.endpoint, "clients", len(p.clients))
	return nil
}

func (p *MCPClientPool) GetClient() (MCPClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return nil, fmt.Errorf("client pool not initialized")
	}

	if len(p.clients) == 0 {
		return nil, fmt.Errorf("no available clients")
	}

	// 简单的轮询策略
	client := p.clients[0]
	p.clients = p.clients[1:]
	return client, nil
}

func (p *MCPClientPool) ReturnClient(client MCPClient) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized && len(p.clients) < p.maxClients {
		p.clients = append(p.clients, client)
	} else {
		client.Close()
	}
}

func (p *MCPClientPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		client.Close()
	}
	p.clients = nil
	p.initialized = false

	p.logger.Info("Client pool closed")
	return nil
}
