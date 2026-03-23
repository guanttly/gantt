package license

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/utils"
)

var (
	logger logging.ILogger
)

// 定义错误代码常量
const (
	maxFailCount       = 3
	verifyInterval     = time.Minute
	LicenseSuccessCode = 0 // 许license验证通过
)

// LicenseStatus 许可证状态
type LicenseStatus struct {
	IsValid      bool      `json:"is_valid"`
	LastCheck    time.Time `json:"last_check"`
	FailCount    int       `json:"fail_count"`
	ErrorMessage string    `json:"error_message"`
	mu           sync.RWMutex
}

// LicenseMonitor 许可证监听器
type LicenseMonitor struct {
	cfg       *config.License
	logger    logging.ILogger
	status    *LicenseStatus
	ctx       context.Context
	cancel    context.CancelFunc
	ticker    *time.Ticker
	callbacks []func(bool, error) // 状态变化回调函数
}

// NewLicenseMonitor 创建新的许可证监听器
func NewLicenseMonitor(cfg *config.License, logger logging.ILogger) *LicenseMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &LicenseMonitor{
		cfg:    cfg,
		logger: logger,
		status: &LicenseStatus{
			IsValid:   false,
			LastCheck: time.Time{},
			FailCount: 0,
		},
		ctx:       ctx,
		cancel:    cancel,
		callbacks: make([]func(bool, error), 0),
	}
}

// AddStatusCallback 添加状态变化回调函数
func (m *LicenseMonitor) AddStatusCallback(callback func(isValid bool, err error)) {
	m.callbacks = append(m.callbacks, callback)
}

// notifyCallbacks 通知所有回调函数
func (m *LicenseMonitor) notifyCallbacks(isValid bool, err error) {
	for _, callback := range m.callbacks {
		go callback(isValid, err) // 异步调用回调，避免阻塞
	}
}

// GetStatus 获取当前许可证状态（线程安全）
func (m *LicenseMonitor) GetStatus() LicenseStatus {
	m.status.mu.RLock()
	defer m.status.mu.RUnlock()

	return LicenseStatus{
		IsValid:      m.status.IsValid,
		LastCheck:    m.status.LastCheck,
		FailCount:    m.status.FailCount,
		ErrorMessage: m.status.ErrorMessage,
	}
}

// updateStatus 更新许可证状态（线程安全）
func (m *LicenseMonitor) updateStatus(isValid bool, err error) {
	m.status.mu.Lock()
	defer m.status.mu.Unlock()

	oldStatus := m.status.IsValid
	m.status.LastCheck = time.Now()

	if err != nil {
		m.status.FailCount++
		m.status.ErrorMessage = err.Error()

		// 连续失败超过最大次数，标记为无效
		if m.status.FailCount >= maxFailCount {
			m.status.IsValid = false
		}
	} else {
		m.status.FailCount = 0
		m.status.IsValid = isValid
		m.status.ErrorMessage = ""
	}

	// 如果状态发生变化，通知回调函数
	if oldStatus != m.status.IsValid {
		m.logger.Debug("许可证状态变化",
			"old_status", oldStatus,
			"new_status", m.status.IsValid,
			"fail_count", m.status.FailCount)

		go m.notifyCallbacks(m.status.IsValid, err)
	}
}

// verifyLicenseStatus 验证许可证状态
func (m *LicenseMonitor) verifyLicenseStatus() {
	m.logger.Debug("开始验证许可证状态")
	response, err := VerifyLicense(m.ctx, m.cfg, m.logger)
	if err != nil {
		m.logger.Warn("许可证验证失败", "error", err)
		m.updateStatus(false, err)
		return
	}

	isValid := response.Code == LicenseSuccessCode
	if !isValid {
		err = fmt.Errorf("许可证验证失败: %s", response.Msg)
	}

	m.updateStatus(isValid, err)
}

type VerifyData struct {
	Feature string `json:"feature"`
}

type VerifyResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

// GenVerifyString 生成用于验证的字符串，格式为 "serverName|timestamp"。
func GenVerifyString(ctx context.Context, cfg *config.License) (string, error) {
	verifyStr := cfg.ServerName + "|" + strconv.FormatInt(time.Now().Unix(), 10)

	// 校验返回消息： serverTag | timeStamp
	encryptStr, err := utils.AESEncryptServerFeature(verifyStr)
	if err != nil {
		logger.Error("failed to encrypt verify string", "error", err)
		return "", fmt.Errorf("failed to encrypt verify string: %w", err)
	}

	return encryptStr, nil
}

// VerifyLicense 校验license，调用远程服务进行验证。
func VerifyLicense(ctx context.Context, cfg *config.License, logger logging.ILogger) (*VerifyResponse, error) {

	verifyStr, err := GenVerifyString(ctx, cfg)
	if err != nil {
		logger.Error("生成校验字符串失败", "error", err)
		return nil, err
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// 将结构体转换为JSON字节数组
	requestData := VerifyData{
		Feature: verifyStr,
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		logger.Error("序列化请求数据失败", "error", err)
		return nil, err
	}

	respBytes, err := utils.SendHTTPRequest(ctx, http.MethodPost, cfg.URL+"/license/verify", headers, requestBody)
	if err != nil {
		logger.Error("license校验失败", "error", err)
		return nil, err
	}

	var resJson VerifyResponse
	err = json.Unmarshal(respBytes, &resJson)
	if err != nil {
		logger.Error("license校验失败", "error", err)
		return &resJson, err
	}

	if resJson.Code != LicenseSuccessCode {
		logger.Error("license校验失败:", "error", err)
		return &resJson, errors.New("license校验失败:" + resJson.Msg)
	}

	logger.Debug("license校验成功", "Info", resJson)
	return &resJson, nil
}

// Start 启动轮询监听器（独立线程）
func (m *LicenseMonitor) Start() {
	m.logger.Debug("启动许可证状态监听器", "interval", verifyInterval)

	// 立即执行一次验证
	go m.verifyLicenseStatus()

	// 创建定时器
	m.ticker = time.NewTicker(verifyInterval)

	// 启动独立的goroutine进行轮询
	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Error("许可证监听器异常退出", "panic", r)
			}
		}()

		for {
			select {
			case <-m.ctx.Done():
				m.logger.Debug("许可证监听器收到停止信号")
				return
			case <-m.ticker.C:
				m.verifyLicenseStatus()
			}
		}
	}()
}

// Stop 停止轮询监听器
func (m *LicenseMonitor) Stop() {
	m.logger.Debug("停止许可证状态监听器")

	if m.ticker != nil {
		m.ticker.Stop()
	}

	if m.cancel != nil {
		m.cancel()
	}
}

// IsValid 检查当前许可证是否有效
func (m *LicenseMonitor) IsValid() bool {
	status := m.GetStatus()
	return status.IsValid
}

// WaitForValidLicense 等待许可证变为有效状态
func (m *LicenseMonitor) WaitForValidLicense(timeout time.Duration) error {
	if m.IsValid() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 创建一个channel来接收状态变化通知
	statusChan := make(chan bool, 1)

	// 添加临时回调
	callback := func(isValid bool, err error) {
		if isValid {
			select {
			case statusChan <- true:
			default:
			}
		}
	}
	m.AddStatusCallback(callback)

	select {
	case <-ctx.Done():
		return fmt.Errorf("等待有效许可证超时")
	case <-statusChan:
		return nil
	}
}

// 全局监听器实例
var globalMonitor *LicenseMonitor
var monitorOnce sync.Once

// InitGlobalMonitor 初始化全局许可证监听器
func InitGlobalMonitor(cfg *config.License, logger logging.ILogger) *LicenseMonitor {
	monitorOnce.Do(func() {
		globalMonitor = NewLicenseMonitor(cfg, logger)
		globalMonitor.Start()
	})
	return globalMonitor
}

// IsGlobalMonitorInitialized 检查全局监听器是否已初始化
func IsGlobalMonitorInitialized() bool {
	return globalMonitor != nil
}

// GetGlobalMonitor 获取全局许可证监听器
func GetGlobalMonitor() *LicenseMonitor {
	return globalMonitor
}

// StopGlobalMonitor 停止全局许可证监听器
func StopGlobalMonitor() {
	if globalMonitor != nil {
		globalMonitor.Stop()
	}
}
