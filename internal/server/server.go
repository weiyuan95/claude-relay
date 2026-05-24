package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/weiyuanlee/claude-code-relay/internal/hook"
	"github.com/weiyuanlee/claude-code-relay/internal/model"
)

type Server struct {
	handler *hook.Handler
	port    int
	mux     *http.ServeMux
}

func New(handler *hook.Handler, port int) *Server {
	s := &Server{
		handler: handler,
		port:    port,
		mux:     http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /permission", s.handlePermission)
	s.mux.HandleFunc("GET /status", s.handleStatus)
	s.mux.HandleFunc("POST /mode", s.handleMode)
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handlePermission(w http.ResponseWriter, r *http.Request) {
	var input model.HookInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("parse request: %v", err), http.StatusBadRequest)
		return
	}

	if input.SessionID == "" {
		input.SessionID = "default"
	}

	decision, err := s.handler.HandleRequest(input)
	if err != nil {
		http.Error(w, fmt.Sprintf("handle request: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.PermissionResponse{
		RequestID: "",
		Decision:  decision,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	pending := s.handler.GetPendingRequests()
	resp := model.StatusResponse{
		Mode:                s.handler.GetMode(),
		PendingRequests:     pending,
		PendingRequestCount: len(pending),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleMode(w http.ResponseWriter, r *http.Request) {
	var req model.ModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("parse request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Mode != model.ModeLocal && req.Mode != model.ModeTelegram {
		http.Error(w, fmt.Sprintf("invalid mode: %s (use 'local' or 'telegram')", req.Mode), http.StatusBadRequest)
		return
	}

	s.handler.SetMode(req.Mode)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"mode":   string(req.Mode),
	})
}

func (s *Server) Addr() string {
	return fmt.Sprintf("127.0.0.1:%d", s.port)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}
