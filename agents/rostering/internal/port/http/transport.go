package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"jusha/agent/rostering/domain/service"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
)

func NewHTTPHandler(svc service.IServiceProvider, logger logging.ILogger) http.Handler {
	r := mux.NewRouter()

	// 健康检查
	r.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet)

	// WebSocket 连接端点
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		infra := svc.GetInfrastructure()
		if infra == nil {
			logger.Error("Infrastructure not available")
			http.Error(w, "Service not ready", http.StatusServiceUnavailable)
			return
		}
		infra.HandleWebSocket(w, r)
	}).Methods(http.MethodGet)

	// 会话管理端点
	sessionRouter := r.PathPrefix("/api/sessions").Subrouter()
	sessionRouter.HandleFunc("", createSessionHandler(svc, logger)).Methods(http.MethodPost)
	sessionRouter.HandleFunc("/{sessionId}", getSessionHandler(svc, logger)).Methods(http.MethodGet)
	sessionRouter.HandleFunc("/{sessionId}/events", sendEventHandler(svc, logger)).Methods(http.MethodPost)
	sessionRouter.HandleFunc("/{sessionId}/messages", sendMessageHandler(svc, logger)).Methods(http.MethodPost)
	// load-conversation 已通过 WebSocket 实现，不再需要 HTTP 端点

	return r
}

// createSessionHandler 创建会话
func createSessionHandler(svc service.IServiceProvider, logger logging.ILogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OrgID    string `json:"orgId"`
			UserID   string `json:"userId"`
			Workflow string `json:"workflow"` // 工作流名称，如 "schedule.create"
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to unmarshal request", "error", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// 创建会话
		sessionSvc := svc.GetSessionService()
		sess, err := sessionSvc.Create(r.Context(), session.CreateSessionRequest{
			OrgID:     req.OrgID,
			UserID:    req.UserID,
			AgentType: req.Workflow,
		})
		if err != nil {
			logger.Error("Failed to create session", "error", err)
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		// 返回会话信息
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"sessionId": sess.ID,
			"orgId":     sess.OrgID,
			"userId":    sess.UserID,
			"workflow":  sess.WorkflowMeta.Workflow,
			"state":     sess.WorkflowMeta.Phase,
		})
	}
}

// getSessionHandler 获取会话信息
func getSessionHandler(svc service.IServiceProvider, logger logging.ILogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sessionID := vars["sessionId"]

		sessionSvc := svc.GetSessionService()
		sess, err := sessionSvc.Get(r.Context(), sessionID)
		if err != nil {
			logger.Error("Session not found", "sessionId", sessionID, "error", err)
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sess)
	}
}

// sendEventHandler 发送工作流事件
func sendEventHandler(svc service.IServiceProvider, logger logging.ILogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sessionID := vars["sessionId"]

		var req struct {
			Event   string `json:"event"`   // 事件名称，如 "_schedule_create_start_"
			Payload any    `json:"payload"` // 事件数据
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to unmarshal request", "error", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// 发送事件到工作流
		infra := svc.GetInfrastructure()
		if infra == nil {
			logger.Error("Infrastructure not available")
			http.Error(w, "Service not ready", http.StatusServiceUnavailable)
			return
		}

		if err := infra.SendEvent(r.Context(), sessionID, engine.Event(req.Event), req.Payload); err != nil {
			logger.Error("Failed to send event", "sessionId", sessionID, "event", req.Event, "error", err)
			http.Error(w, "Failed to send event", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"event":  req.Event,
		})
	}
}

// sendMessageHandler 发送消息到会话（用于用户输入）
func sendMessageHandler(svc service.IServiceProvider, logger logging.ILogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sessionID := vars["sessionId"]

		var req struct {
			Message string `json:"message"` // 用户消息
			Data    any    `json:"data"`    // 附加数据
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to unmarshal request", "error", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// 通过 Bridge 广播到会话
		infra := svc.GetInfrastructure()
		if infra == nil {
			logger.Error("Infrastructure not available")
			http.Error(w, "Service not ready", http.StatusServiceUnavailable)
			return
		}

		bridge := infra.GetBridge()
		if err := bridge.BroadcastToSession(sessionID, "message", map[string]any{
			"message": req.Message,
			"data":    req.Data,
		}); err != nil {
			logger.Error("Failed to broadcast message", "sessionId", sessionID, "error", err)
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}
