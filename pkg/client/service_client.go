package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/model"

	"github.com/mitchellh/mapstructure"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	nacos_model "github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

const default_cluster = "DEFAULT"

// CircuitState 熔断器状态
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type ServiceName string

const (
	ServiceName_IngestionService  ServiceName = "ingestion-service"
	ServiceName_QueryService      ServiceName = "query-service"
	ServiceName_ManagementService ServiceName = "management-service"
)

var (
	serviceInstances      = map[string]nacos_model.Instance{}
	serviceInstancesMutex sync.RWMutex
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	mutex            sync.RWMutex
	state            CircuitState
	failureCount     int
	successCount     int
	lastFailTime     time.Time
	failureThreshold int
	recoveryTimeout  time.Duration
	successThreshold int
}

// NewCircuitBreaker 创建新的熔断器
func NewCircuitBreaker(failureThreshold int, recoveryTimeout time.Duration, successThreshold int) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		recoveryTimeout:  recoveryTimeout,
		successThreshold: successThreshold,
	}
}

// CanExecute 检查是否可以执行请求
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailTime) > cb.recoveryTimeout {
			cb.mutex.RUnlock()
			cb.mutex.Lock()
			if cb.state == StateOpen && time.Since(cb.lastFailTime) > cb.recoveryTimeout {
				cb.state = StateHalfOpen
				cb.successCount = 0
			}
			cb.mutex.Unlock()
			cb.mutex.RLock()
			return cb.state == StateHalfOpen
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// OnSuccess 记录成功
func (cb *CircuitBreaker) OnSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount = 0
	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
		}
	}
}

// OnFailure 记录失败
func (cb *CircuitBreaker) OnFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount++
	cb.lastFailTime = time.Now()

	if cb.state == StateClosed && cb.failureCount >= cb.failureThreshold {
		cb.state = StateOpen
	} else if cb.state == StateHalfOpen {
		cb.state = StateOpen
	}
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 100 * time.Millisecond,
	MaxDelay:     2 * time.Second,
	Multiplier:   2.0,
}

// ServiceClient 通用服务调用客户端
type ServiceClient struct {
	nacosClient    naming_client.INamingClient
	logger         logging.ILogger
	client         *http.Client
	circuitBreaker *CircuitBreaker
	retryConfig    RetryConfig

	ServiceName  string // 服务名
	ServiceGroup string // 服务组名（可选）
}

// ServiceClientConfig 服务客户端配置
type ServiceClientConfig struct {
	Timeout        time.Duration
	RetryConfig    *RetryConfig
	CircuitBreaker *CircuitBreaker
}

// NewServiceClient 创建新的服务客户端
func NewServiceClient(
	nacosClient naming_client.INamingClient,
	serviceName ServiceName,
	logger logging.ILogger,
) *ServiceClient {
	return NewServiceClientWithConfig(
		nacosClient,
		string(serviceName),
		"DEFAULT_GROUP", // 默认组名
		ServiceClientConfig{
			Timeout:        30 * time.Second,
			RetryConfig:    &DefaultRetryConfig,
			CircuitBreaker: NewCircuitBreaker(5, 30*time.Second, 3),
		},
		logger,
	)
}

// NewServiceClientWithConfig 使用配置创建服务客户端
func NewServiceClientWithConfig(
	nacosClient naming_client.INamingClient,
	serviceName string,
	serviceGroup string,
	config ServiceClientConfig,
	logger logging.ILogger,
) *ServiceClient {
	if nacosClient == nil {
		panic("nacosClient cannot be nil")
	}

	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	retryConfig := DefaultRetryConfig
	if config.RetryConfig != nil {
		retryConfig = *config.RetryConfig
	}

	circuitBreaker := config.CircuitBreaker
	if circuitBreaker == nil {
		circuitBreaker = NewCircuitBreaker(5, 30*time.Second, 3)
	}

	sc := &ServiceClient{
		nacosClient:    nacosClient,
		client:         client,
		circuitBreaker: circuitBreaker,
		retryConfig:    retryConfig,
		logger:         logger.With("component", "ServiceClient", "serviceName", serviceName, "serviceGroup", serviceGroup),

		ServiceName:  serviceName,
		ServiceGroup: serviceGroup,
	}

	subscribeParam := &vo.SubscribeParam{
		ServiceName:       serviceName,
		SubscribeCallback: sc.subscribeCallback,
		Clusters:          []string{default_cluster},
		GroupName:         default_group,
	}
	nacosClient.Subscribe(subscribeParam)
	return sc
}

// CallRequest 请求参数
type CallRequest struct {
	Method  string
	Path    string
	Body    any
	Headers map[string]string
	// 可选的重试配置，如果不设置则使用客户端默认配置
	RetryConfig *RetryConfig
}

// CallResponse 响应结果
type CallResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Call 执行服务调用
func (c *ServiceClient) Call(ctx context.Context, req CallRequest) (*CallResponse, error) {
	// 检查熔断器状态
	if !c.circuitBreaker.CanExecute() {
		return nil, errors.New("service circuit breaker is open")
	}

	retryConfig := c.retryConfig
	if req.RetryConfig != nil {
		retryConfig = *req.RetryConfig
	}

	var lastErr error
	var lastResp *CallResponse
	delay := retryConfig.InitialDelay

	for attempt := 1; attempt <= retryConfig.MaxAttempts; attempt++ {
		resp, err := c.executeCall(ctx, req)
		if err == nil && resp.StatusCode < 500 {
			// 成功或客户端错误（4xx），不需要重试
			c.circuitBreaker.OnSuccess()
			return resp, nil
		}

		lastErr = err
		lastResp = resp
		if err != nil {
			c.circuitBreaker.OnFailure()
		}

		// 检查是否应该重试
		if attempt == retryConfig.MaxAttempts {
			break
		}

		// 检查错误类型，某些错误不应该重试
		if !shouldRetry(err, resp) {
			break
		}

		// 等待重试
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}

		// 计算下次重试的延迟时间
		delay = time.Duration(float64(delay) * retryConfig.Multiplier)
		if delay > retryConfig.MaxDelay {
			delay = retryConfig.MaxDelay
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("service call failed after %d attempts: %w", retryConfig.MaxAttempts, lastErr)
	}

	// 如果有响应但是5xx错误
	if lastResp != nil {
		c.circuitBreaker.OnFailure()
		return lastResp, nil
	}

	// 这种情况理论上不应该发生
	return nil, errors.New("service call completed without error or response")
}

// executeCall 执行单次调用
func (c *ServiceClient) executeCall(ctx context.Context, req CallRequest) (*CallResponse, error) {
	if c.nacosClient == nil {
		return nil, errors.New("service nacosClient not initialized")
	}

	// 获取服务端点
	instance, err := c.GetInstance()
	if err != nil {
		return nil, fmt.Errorf("no available service instance: %w", err)
	}

	targetUrl, err := getServiceUrl(instance)
	if err != nil {
		c.logger.ErrorContext(ctx, "获取服务端点失败", "error", err)
		return nil, fmt.Errorf("failed to get service endpoint: %w", err)
	}

	// 构建请求URL
	reqUrl := fmt.Sprintf("%s%s", targetUrl.String(), req.Path)

	// 准备请求体
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, reqUrl, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置默认头部
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// 设置自定义头部
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// 执行请求
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &CallResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}, nil
}

// shouldRetry 判断是否应该重试
func shouldRetry(err error, resp *CallResponse) bool {
	// 网络错误应该重试
	if err != nil {
		// 上下文取消不应该重试
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		return true
	}

	// 5xx 服务器错误应该重试
	if resp != nil && resp.StatusCode >= 500 {
		return true
	}

	// 其他情况不重试
	return false
}

func (c *ServiceClient) GetInstance() (*nacos_model.Instance, error) {
	if c.nacosClient == nil {
		return nil, errors.New("nacos client is not initialized")
	}

	instance, err := c.nacosClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: c.ServiceName,
		GroupName:   c.ServiceGroup,
		Clusters:    []string{"DEFAULT"},
	})
	if err != nil {
		c.logger.Error("获取服务实例失败", "error", err)
		return nil, fmt.Errorf("failed to select service instance: %w", err)
	}
	return instance, nil
}

func (c *ServiceClient) GetUrl() (*url.URL, error) {
	instance, err := c.GetInstance()
	if err != nil {
		return nil, err
	}
	return getServiceUrl(instance)
}

func (c *ServiceClient) GetServiceName() ServiceName {
	return ServiceName(c.ServiceName)
}

func (c *ServiceClient) GetServiceGroup() string {
	return c.ServiceGroup
}

func (c *ServiceClient) GetIP() string {
	instance, err := c.GetInstance()
	if err != nil {
		return ""
	}
	return instance.Ip
}

func (c *ServiceClient) GetPort() uint64 {
	instance, err := c.GetInstance()
	if err != nil {
		return 0
	}
	return instance.Port
}

// CallAndExpectOK 执行调用并期望200状态码
func (c *ServiceClient) CallAndExpectOK(ctx context.Context, req CallRequest) error {
	resp, err := c.Call(ctx, req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service call failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	var response model.ApiResponse
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		return nil
	}

	if response.Code != 0 {
		return fmt.Errorf("service call failed with code %d: %s", response.Code, response.Message)
	}

	return nil
}

// CallAndDecodeJSON 执行调用并解码JSON响应
func (c *ServiceClient) CallAndDecodeJSON(ctx context.Context, req CallRequest, result any) error {
	resp, err := c.Call(ctx, req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service call failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	// 直接解析到 ApiResponse，使用 json.RawMessage 来处理 Data 字段
	var response model.ApiResponse
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Code != 0 {
		c.logger.ErrorContext(ctx, "Service call failed", "code", response.Code, "message", response.Message, "request", req)
		return fmt.Errorf("service call failed with code %d: %s", response.Code, response.Message)
	}
	if response.Data == nil {
		return errors.New("service call returned nil data")
	}

	if err := mapstructure.Decode(response.Data, result); err != nil {
		return fmt.Errorf("failed to decode response data: %w", err)
	}
	return nil
}

// GetCircuitBreakerState 获取熔断器状态
func (c *ServiceClient) GetCircuitBreakerState() CircuitState {
	c.circuitBreaker.mutex.RLock()
	defer c.circuitBreaker.mutex.RUnlock()
	return c.circuitBreaker.state
}

// GET 执行GET请求并解码JSON响应
func (c *ServiceClient) Get(ctx context.Context, path string, result any) error {
	req := CallRequest{
		Method: http.MethodGet,
		Path:   path,

		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return c.CallAndDecodeJSON(ctx, req, result)
}

// GetAndExpectOK 执行GET请求并期望200状态码
func (c *ServiceClient) GetAndExpectOK(ctx context.Context, path string) error {
	req := CallRequest{
		Method: http.MethodGet,
		Path:   path,

		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return c.CallAndExpectOK(ctx, req)
}

// Post 执行POST请求并解码JSON响应
func (c *ServiceClient) Post(ctx context.Context, path string, body any, result any) error {
	req := CallRequest{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,

		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return c.CallAndDecodeJSON(ctx, req, result)
}

// PostAndExpectOK 执行POST请求并期望200状态码
func (c *ServiceClient) PostAndExpectOK(ctx context.Context, path string, body any) error {
	req := CallRequest{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,

		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return c.CallAndExpectOK(ctx, req)
}

func (c *ServiceClient) subscribeCallback(services []nacos_model.Instance, err error) {
	// 每次变更都重新订阅，以期刷新缓存，解决Nacos的SDK缓存BUG
	// 1.判断services是否变更
	if err != nil {
		c.logger.Error("订阅服务失败", "error", err)
		return
	}

	serviceInstancesMutex.RLock()
	if len(services) == len(serviceInstances) {
		same := true
		for _, instance := range services {
			serviceKey := fmt.Sprintf("%s:%d", instance.Ip, instance.Port)
			if _, exists := serviceInstances[serviceKey]; !exists {
				same = false
				break
			}
		}
		if same {
			serviceInstancesMutex.RUnlock()
			c.logger.Info("服务实例未变更，跳过订阅更新")
			return
		}
	}
	serviceInstancesMutex.RUnlock()

	// 2.更新serviceInstances
	c.logger.Info("服务实例变更，更新订阅", "count", len(services))
	serviceInstancesMutex.Lock()
	serviceInstances = make(map[string]nacos_model.Instance)
	for _, instance := range services {
		serviceKey := fmt.Sprintf("%s:%d", instance.Ip, instance.Port)
		serviceInstances[serviceKey] = instance
	}
	serviceInstancesMutex.Unlock()

	// 3.重新订阅
	c.logger.Info("重新订阅服务实例", "serviceName", c.ServiceName, "count", len(serviceInstances))
	subscribeParam := &vo.SubscribeParam{
		ServiceName:       c.ServiceName,
		SubscribeCallback: c.subscribeCallback,
		Clusters:          []string{default_cluster},
		GroupName:         default_group,
	}
	c.nacosClient.Unsubscribe(subscribeParam)
	c.nacosClient.Subscribe(subscribeParam)
}

func getServiceUrl(instance *nacos_model.Instance) (*url.URL, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}
	urlStr := fmt.Sprintf("%s:%d", instance.Ip, instance.Port)
	if instance.Metadata != nil {
		if instance.Metadata["url"] != "" {
			urlStr = instance.Metadata["url"]
		}
	}
	if !hasScheme(urlStr) {
		urlStr = "http://" + urlStr
	}
	return url.Parse(urlStr)
}

// hasScheme 判断字符串是否包含 scheme
func hasScheme(addr string) bool {
	return len(addr) > 7 && (addr[:7] == "http://" || addr[:8] == "https://")
}
