package handler

import (
	"encoding/json"
	"net/http"

	"github.com/junbin-yang/go-kitbox/examples/app_framework/internal/service"
	"github.com/junbin-yang/go-kitbox/pkg/logger"
)

// APIHandler API处理器
type APIHandler struct {
	svc    *service.DemoService
	logger *logger.Logger
}

// NewAPIHandler 创建API处理器
func NewAPIHandler(svc *service.DemoService, log *logger.Logger) *APIHandler {
	return &APIHandler{
		svc:    svc,
		logger: log,
	}
}

// ServeHTTP 处理API请求
func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result := h.svc.Process(r.Context())

	h.logger.Info("处理API请求",
		logger.String("method", r.Method),
		logger.String("path", r.URL.Path),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": result,
		"message": "success",
	})
}
