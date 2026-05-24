package model

import "time"

type Mode string

const (
	ModeLocal    Mode = "local"
	ModeTelegram Mode = "telegram"
)

type PermissionRequest struct {
	RequestID    string    `json:"request_id"`
	SessionID    string    `json:"session_id"`
	ToolName     string    `json:"tool_name"`
	Description  string    `json:"description"`
	InputPreview string    `json:"input_preview"`
	CreatedAt    time.Time `json:"created_at"`
}

type PermissionDecision string

const (
	DecisionAllow PermissionDecision = "allow"
	DecisionDeny  PermissionDecision = "deny"
	DecisionLocal PermissionDecision = "local"
)

type PermissionResponse struct {
	RequestID string             `json:"request_id"`
	Decision  PermissionDecision `json:"decision"`
}

type HookInput struct {
	ToolName     string `json:"tool_name"`
	Description  string `json:"description"`
	InputPreview string `json:"input_preview"`
	SessionID    string `json:"session_id"`
}

type HookOutput struct {
	HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

type HookSpecificOutput struct {
	HookEventName string   `json:"hookEventName"`
	Decision      *Decision `json:"decision,omitempty"`
}

type Decision struct {
	Behavior string `json:"behavior"`
}

type StatusResponse struct {
	Mode                Mode                `json:"mode"`
	PendingRequests     []PermissionRequest `json:"pending_requests"`
	PendingRequestCount int                 `json:"pending_request_count"`
}

type ModeRequest struct {
	Mode Mode `json:"mode"`
}
