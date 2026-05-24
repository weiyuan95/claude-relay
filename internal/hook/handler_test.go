package hook

import (
	"sync"
	"testing"
	"time"

	"github.com/weiyuanlee/claude-code-relay/internal/model"
)

type mockNotifier struct {
	mu            sync.Mutex
	requests      []model.PermissionRequest
	notifications []string
}

func (m *mockNotifier) SendPermissionRequest(req model.PermissionRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, req)
	return nil
}

func (m *mockNotifier) SendNotification(msg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, msg)
	return nil
}

func (m *mockNotifier) GetRequests() []model.PermissionRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requests
}

func TestLocalMode_ReturnsLocal(t *testing.T) {
	notifier := &mockNotifier{}
	h := NewHandler(model.ModeLocal, 5*time.Second, false, notifier)

	decision, err := h.HandleRequest(model.HookInput{
		ToolName:     "Bash",
		InputPreview: "git status",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision != model.DecisionLocal {
		t.Errorf("expected DecisionLocal, got %s", decision)
	}
}

func TestLocalMode_SendsNotification(t *testing.T) {
	notifier := &mockNotifier{}
	h := NewHandler(model.ModeLocal, 5*time.Second, true, notifier)

	h.HandleRequest(model.HookInput{ToolName: "Bash", InputPreview: "git status"})

	time.Sleep(50 * time.Millisecond)

	if len(notifier.GetRequests()) > 0 {
		t.Error("should not send permission request in local mode")
	}
}

func TestTelegramMode_ResolvesAllow(t *testing.T) {
	notifier := &mockNotifier{}
	h := NewHandler(model.ModeTelegram, 5*time.Second, false, notifier)

	var decision model.PermissionDecision
	var err error
	done := make(chan struct{})

	go func() {
		decision, err = h.HandleRequest(model.HookInput{
			ToolName:     "Bash",
			InputPreview: "git push",
		})
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	reqs := notifier.GetRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}

	h.ResolveRequest(reqs[0].RequestID, model.DecisionAllow)

	<-done
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision != model.DecisionAllow {
		t.Errorf("expected DecisionAllow, got %s", decision)
	}
}

func TestTelegramMode_ResolvesDeny(t *testing.T) {
	notifier := &mockNotifier{}
	h := NewHandler(model.ModeTelegram, 5*time.Second, false, notifier)

	var decision model.PermissionDecision
	done := make(chan struct{})

	go func() {
		decision, _ = h.HandleRequest(model.HookInput{
			ToolName:     "Bash",
			InputPreview: "rm -rf /",
		})
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	reqs := notifier.GetRequests()
	h.ResolveRequest(reqs[0].RequestID, model.DecisionDeny)

	<-done
	if decision != model.DecisionDeny {
		t.Errorf("expected DecisionDeny, got %s", decision)
	}
}

func TestTelegramMode_Timeout(t *testing.T) {
	notifier := &mockNotifier{}
	h := NewHandler(model.ModeTelegram, 100*time.Millisecond, false, notifier)

	decision, err := h.HandleRequest(model.HookInput{
		ToolName:     "Bash",
		InputPreview: "git push",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision != model.DecisionLocal {
		t.Errorf("expected DecisionLocal on timeout, got %s", decision)
	}
}

func TestTelegramMode_InvalidResolve(t *testing.T) {
	notifier := &mockNotifier{}
	h := NewHandler(model.ModeTelegram, 5*time.Second, false, notifier)

	err := h.ResolveRequest("nonexistent", model.DecisionAllow)
	if err == nil {
		t.Error("expected error for nonexistent request")
	}
}

func TestSetMode(t *testing.T) {
	h := NewHandler(model.ModeLocal, 5*time.Second, false, nil)

	if h.GetMode() != model.ModeLocal {
		t.Error("expected default mode local")
	}

	h.SetMode(model.ModeTelegram)
	if h.GetMode() != model.ModeTelegram {
		t.Error("expected mode telegram after set")
	}
}

func TestSetNotifier(t *testing.T) {
	h := NewHandler(model.ModeLocal, 5*time.Second, true, nil)

	notifier := &mockNotifier{}
	h.SetNotifier(notifier)

	h.HandleRequest(model.HookInput{ToolName: "Bash", InputPreview: "test"})
	time.Sleep(50 * time.Millisecond)

	if len(notifier.notifications) != 1 {
		t.Errorf("expected 1 notification after SetNotifier, got %d", len(notifier.notifications))
	}
}
