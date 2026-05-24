package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/weiyuanlee/claude-code-relay/internal/hook"
	"github.com/weiyuanlee/claude-code-relay/internal/model"
)

func newTestServer() *Server {
	h := hook.NewHandler(model.ModeLocal, 5*time.Second, false, nil)
	return New(h, 7654)
}

func TestPermission_LocalMode(t *testing.T) {
	srv := newTestServer()

	body, _ := json.Marshal(model.HookInput{
		ToolName:     "Bash",
		InputPreview: "git status",
	})
	req := httptest.NewRequest("POST", "/permission", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handlePermission(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp model.PermissionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Decision != model.DecisionLocal {
		t.Errorf("expected DecisionLocal, got %s", resp.Decision)
	}
}

func TestStatus(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()

	srv.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp model.StatusResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Mode != model.ModeLocal {
		t.Errorf("expected ModeLocal, got %s", resp.Mode)
	}
}

func TestSetMode(t *testing.T) {
	srv := newTestServer()

	body, _ := json.Marshal(model.ModeRequest{Mode: model.ModeTelegram})
	req := httptest.NewRequest("POST", "/mode", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleMode(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if srv.handler.GetMode() != model.ModeTelegram {
		t.Errorf("expected ModeTelegram after set")
	}
}

func TestSetMode_InvalidMode(t *testing.T) {
	srv := newTestServer()

	body, _ := json.Marshal(model.ModeRequest{Mode: "invalid"})
	req := httptest.NewRequest("POST", "/mode", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPermission_BadJSON(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("POST", "/permission", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handlePermission(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
