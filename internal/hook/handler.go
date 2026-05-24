package hook

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/weiyuanlee/claude-code-relay/internal/model"
)

type Notifier interface {
	SendPermissionRequest(req model.PermissionRequest) error
	SendNotification(message string) error
}

type Handler struct {
	mu            sync.RWMutex
	mode          model.Mode
	pending       map[string]chan model.PermissionDecision
	requests      map[string]model.PermissionRequest
	timeout       time.Duration
	notifyInLocal bool
	notifier      Notifier
}

func NewHandler(mode model.Mode, timeout time.Duration, notifyInLocal bool, notifier Notifier) *Handler {
	return &Handler{
		mode:          mode,
		pending:       make(map[string]chan model.PermissionDecision),
		requests:      make(map[string]model.PermissionRequest),
		timeout:       timeout,
		notifyInLocal: notifyInLocal,
		notifier:      notifier,
	}
}

func (h *Handler) HandleRequest(ctx context.Context, input model.HookInput) (model.PermissionDecision, error) {
	reqID := uuid.New().String()[:8]
	req := model.PermissionRequest{
		RequestID:    reqID,
		SessionID:    input.SessionID,
		ToolName:     input.ToolName,
		Description:  input.Description,
		InputPreview: input.InputPreview,
		CreatedAt:    time.Now(),
	}

	h.mu.RLock()
	currentMode := h.mode
	h.mu.RUnlock()

	if currentMode == model.ModeLocal {
		if h.notifyInLocal && h.notifier != nil {
			msg := fmt.Sprintf("[local mode] Permission requested\nTool: %s\nCommand: %s", req.ToolName, req.InputPreview)
			go h.notifier.SendNotification(msg)
		}
		return model.DecisionLocal, nil
	}

	ch := make(chan model.PermissionDecision, 1)

	h.mu.Lock()
	h.pending[reqID] = ch
	h.requests[reqID] = req
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.pending, reqID)
		delete(h.requests, reqID)
		h.mu.Unlock()
	}()

	if h.notifier != nil {
		if err := h.notifier.SendPermissionRequest(req); err != nil {
			return model.DecisionLocal, fmt.Errorf("send to telegram: %w", err)
		}
	}

	select {
	case decision := <-ch:
		return decision, nil
	case <-ctx.Done():
		if h.notifier != nil {
			msg := fmt.Sprintf("Request %s cancelled (approved locally)", reqID)
			go h.notifier.SendNotification(msg)
		}
		return model.DecisionLocal, nil
	case <-time.After(h.timeout):
		if h.notifier != nil {
			msg := fmt.Sprintf("Request %s timed out, fell back to local prompt", reqID)
			go h.notifier.SendNotification(msg)
		}
		return model.DecisionLocal, nil
	}
}

func (h *Handler) ResolveRequest(requestID string, decision model.PermissionDecision) error {
	h.mu.RLock()
	ch, ok := h.pending[requestID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no pending request with id %s", requestID)
	}

	ch <- decision
	return nil
}

func (h *Handler) SetMode(mode model.Mode) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.mode = mode
}

func (h *Handler) GetMode() model.Mode {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.mode
}

func (h *Handler) GetPendingRequests() []model.PermissionRequest {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]model.PermissionRequest, 0, len(h.requests))
	for _, req := range h.requests {
		result = append(result, req)
	}
	return result
}

func (h *Handler) SetNotifier(n Notifier) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.notifier = n
}
