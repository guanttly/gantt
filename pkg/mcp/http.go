package mcp

import (
	"context"
	"io"
	"jusha/mcp/pkg/logging"
	"net/http"
	"time"
)

// HTTPTransport HTTP传输实现
type HTTPTransport struct {
	server  MCPServer
	logger  logging.ILogger
	timeout time.Duration
}

func NewHTTPTransport(server MCPServer, timeout time.Duration, logger logging.ILogger) *HTTPTransport {
	return &HTTPTransport{
		server:  server,
		logger:  logger,
		timeout: timeout,
	}
}

func (t *HTTPTransport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(r.Context(), t.timeout)
	defer cancel()

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.logger.Error("Failed to read request body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	t.logger.Debug("Received HTTP request", "method", r.Method, "path", r.URL.Path, "body_size", len(body))

	// 处理MCP消息
	response, err := t.server.HandleMessage(ctx, body)
	if err != nil {
		t.logger.Error("Failed to handle MCP message", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 写入响应
	if _, err := w.Write(response); err != nil {
		t.logger.Error("Failed to write response", "error", err)
		return
	}

	t.logger.Debug("HTTP request handled successfully")
}

// HTTPServer HTTP服务器包装器
type HTTPServer struct {
	server    *http.Server
	transport *HTTPTransport
	logger    logging.ILogger
}

func NewHTTPServer(addr string, mcpServer MCPServer, timeout time.Duration, logger logging.ILogger) *HTTPServer {
	transport := NewHTTPTransport(mcpServer, timeout, logger)

	mux := http.NewServeMux()
	mux.Handle("/mcp", transport)

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return &HTTPServer{
		server:    httpServer,
		transport: transport,
		logger:    logger,
	}
}

func (s *HTTPServer) Start(ctx context.Context) error {
	s.logger.Info("Starting HTTP server", "addr", s.server.Addr)

	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(shutdownCtx)
	}()

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	s.logger.Info("HTTP server stopped")
	return nil
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
