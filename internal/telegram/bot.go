package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	telebot "gopkg.in/telebot.v3"

	"github.com/weiyuanlee/claude-code-relay/internal/model"
)

// HandlerInterface decouples Bot from the concrete hook.Handler,
// breaking the circular dependency (Handler needs Notifier, Bot needs Handler).
type HandlerInterface interface {
	GetMode() model.Mode
	SetMode(mode model.Mode)
	GetPendingRequests() []model.PermissionRequest
	ResolveRequest(requestID string, decision model.PermissionDecision) error
}

type Bot struct {
	bot       *telebot.Bot
	handler   HandlerInterface
	allowedID int64
	serverURL string
}

func NewBot(token string, allowedID int64, handler HandlerInterface, serverURL string) (*Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	tg := &Bot{
		bot:       b,
		handler:   handler,
		allowedID: allowedID,
		serverURL: serverURL,
	}

	tg.registerHandlers()
	return tg, nil
}

func (tg *Bot) registerHandlers() {
	tg.bot.Handle("/start", tg.auth(tg.handleStart))
	tg.bot.Handle("/mode", tg.auth(tg.handleMode))
	tg.bot.Handle("/status", tg.auth(tg.handleStatus))
	tg.bot.Handle("/allow", tg.auth(tg.handleAllow))
	tg.bot.Handle("/deny", tg.auth(tg.handleDeny))

	// Inline button callback
	tg.bot.Handle(telebot.OnCallback, tg.auth(tg.handleCallback))
}

func (tg *Bot) auth(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != tg.allowedID {
			return c.Reply("Unauthorized.")
		}
		return next(c)
	}
}

func (tg *Bot) handleStart(c telebot.Context) error {
	mode := tg.handler.GetMode()
	return c.Reply(fmt.Sprintf("Claude Code Relay bot ready.\n\nCurrent mode: %s\n\nCommands:\n/mode local — local prompts only\n/mode telegram — approve via Telegram\n/status — show current state", mode))
}

func (tg *Bot) handleMode(c telebot.Context) error {
	args := strings.TrimSpace(c.Message().Payload)
	if args == "" {
		mode := tg.handler.GetMode()
		return c.Reply(fmt.Sprintf("Current mode: %s\nUsage: /mode local or /mode telegram", mode))
	}

	var newMode model.Mode
	switch strings.ToLower(args) {
	case "local":
		newMode = model.ModeLocal
	case "telegram":
		newMode = model.ModeTelegram
	default:
		return c.Reply(fmt.Sprintf("Unknown mode: %s\nUse 'local' or 'telegram'", args))
	}

	tg.handler.SetMode(newMode)
	return c.Reply(fmt.Sprintf("Mode set to: %s", newMode))
}

func (tg *Bot) handleStatus(c telebot.Context) error {
	mode := tg.handler.GetMode()
	pending := tg.handler.GetPendingRequests()

	msg := fmt.Sprintf("Mode: %s\nPending requests: %d", mode, len(pending))
	if len(pending) > 0 {
		msg += "\n\n"
		for _, req := range pending {
			msg += fmt.Sprintf("• [%s] %s: %s\n", req.RequestID, req.ToolName, truncate(req.InputPreview, 50))
		}
	}

	return c.Reply(msg)
}

func (tg *Bot) handleAllow(c telebot.Context) error {
	reqID := strings.TrimSpace(c.Message().Payload)
	if reqID == "" {
		return c.Reply("Usage: /allow <request_id>")
	}

	if err := tg.handler.ResolveRequest(reqID, model.DecisionAllow); err != nil {
		return c.Reply(fmt.Sprintf("Error: %v", err))
	}
	return c.Reply(fmt.Sprintf("Approved request %s", reqID))
}

func (tg *Bot) handleDeny(c telebot.Context) error {
	reqID := strings.TrimSpace(c.Message().Payload)
	if reqID == "" {
		return c.Reply("Usage: /deny <request_id>")
	}

	if err := tg.handler.ResolveRequest(reqID, model.DecisionDeny); err != nil {
		return c.Reply(fmt.Sprintf("Error: %v", err))
	}
	return c.Reply(fmt.Sprintf("Denied request %s", reqID))
}

func (tg *Bot) handleCallback(c telebot.Context) error {
	data := c.Data()
	// Callback data format: "allow:<id>" or "deny:<id>"
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return c.Respond(&telebot.CallbackResponse{Text: "Invalid callback"})
	}

	action, reqID := parts[0], parts[1]
	var decision model.PermissionDecision

	switch action {
	case "allow":
		decision = model.DecisionAllow
	case "deny":
		decision = model.DecisionDeny
	default:
		return c.Respond(&telebot.CallbackResponse{Text: "Unknown action"})
	}

	if err := tg.handler.ResolveRequest(reqID, decision); err != nil {
		return c.Respond(&telebot.CallbackResponse{Text: fmt.Sprintf("Error: %v", err)})
	}

	// Update the message to show the decision
	emoji := "✅"
	if decision == model.DecisionDeny {
		emoji = "❌"
	}

	c.Edit(fmt.Sprintf("%s Request %s — %s", emoji, reqID, decision))
	return c.Respond(&telebot.CallbackResponse{Text: fmt.Sprintf("%s %s", emoji, decision)})
}

func (tg *Bot) SendPermissionRequest(req model.PermissionRequest) error {
	text := fmt.Sprintf("Permission Request [%s]\n\nTool: %s\nCommand: %s\n\nID: %s",
		req.SessionID, req.ToolName, req.InputPreview, req.RequestID)

	selector := &telebot.ReplyMarkup{}
	btnAllow := selector.Data("✅ Allow", "allow:"+req.RequestID)
	btnDeny := selector.Data("❌ Deny", "deny:"+req.RequestID)
	selector.Inline(
		selector.Row(btnAllow, btnDeny),
	)

	_, err := tg.bot.Send(telebot.ChatID(tg.allowedID), text, selector)
	return err
}

func (tg *Bot) SendNotification(message string) error {
	_, err := tg.bot.Send(telebot.ChatID(tg.allowedID), message)
	return err
}

func (tg *Bot) Start() {
	log.Println("Telegram bot started")
	tg.bot.Start()
}

func (tg *Bot) Stop() {
	tg.bot.Stop()
}

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}

// ParseUserID parses a Telegram user ID from string for config validation.
func ParseUserID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
