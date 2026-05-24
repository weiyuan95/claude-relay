package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
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
	bot         *telebot.Bot
	handler     HandlerInterface
	allowedID   int64
	serverURL   string
	summaries   map[string]string
	messages    map[string]*telebot.Message
	summariesMu sync.RWMutex
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
		summaries: make(map[string]string),
		messages:  make(map[string]*telebot.Message),
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

	// Debug: show what we received
	log.Printf("callback data: %q (len=%d)", data, len(data))

	// Strip telebot internal prefix if present
	data = strings.TrimPrefix(data, "\f")

	// Try colon separator first (three-arg Data form), then underscore (two-arg form)
	var action, reqID string
	if parts := strings.SplitN(data, ":", 2); len(parts) == 2 && (parts[0] == "allow" || parts[0] == "deny") {
		action, reqID = parts[0], parts[1]
	} else if parts := strings.SplitN(data, "_", 2); len(parts) == 2 && (parts[0] == "allow" || parts[0] == "deny") {
		action, reqID = parts[0], parts[1]
	} else {
		return c.Respond(&telebot.CallbackResponse{Text: fmt.Sprintf("Invalid: %q", data)})
	}
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

	summary := tg.getSummary(reqID)

	emoji := "✅"
	label := "Allowed"
	if decision == model.DecisionDeny {
		emoji = "❌"
		label = "Denied"
	}

	c.Edit(fmt.Sprintf("%s <b>%s</b> — %s", emoji, summary, label), telebot.ModeHTML)
	return c.Respond(&telebot.CallbackResponse{Text: fmt.Sprintf("%s %s", emoji, label)})
}

func (tg *Bot) SendPermissionRequest(req model.PermissionRequest) error {
	summary := fmt.Sprintf("%s: %s", req.ToolName, truncate(req.InputPreview, 40))
	tg.saveSummary(req.RequestID, summary)

	cmd := truncate(req.InputPreview, 200)
	text := fmt.Sprintf("🔐 <b>Permission Request</b>\n\n<b>Tool:</b> %s\n<b>Command:</b>\n<code>%s</code>",
		req.ToolName, cmd)

	selector := &telebot.ReplyMarkup{}
	btnAllow := selector.Data("✅ Allow", "allow_"+req.RequestID)
	btnDeny := selector.Data("❌ Deny", "deny_"+req.RequestID)
	selector.Inline(
		selector.Row(btnAllow, btnDeny),
	)

	msg, err := tg.bot.Send(telebot.ChatID(tg.allowedID), text, selector, telebot.ModeHTML)
	if err != nil {
		return err
	}
	tg.saveMessage(req.RequestID, msg)
	return nil
}

func (tg *Bot) SendNotification(message string) error {
	_, err := tg.bot.Send(telebot.ChatID(tg.allowedID), message)
	return err
}

func (tg *Bot) CancelRequest(requestID string) {
	summary := tg.getSummary(requestID)
	msg := tg.getMessage(requestID)
	if msg == nil {
		return
	}
	tg.bot.Edit(msg, fmt.Sprintf("👌 <b>%s</b> — handled locally", summary), telebot.ModeHTML)
}

func (tg *Bot) Start() {
	log.Println("Telegram bot started")
	tg.bot.Start()
}

func (tg *Bot) Stop() {
	tg.bot.Stop()
}

func (tg *Bot) saveSummary(reqID, summary string) {
	tg.summariesMu.Lock()
	tg.summaries[reqID] = summary
	tg.summariesMu.Unlock()
}

func (tg *Bot) getSummary(reqID string) string {
	tg.summariesMu.RLock()
	defer tg.summariesMu.RUnlock()
	if s, ok := tg.summaries[reqID]; ok {
		return s
	}
	return reqID
}

func (tg *Bot) saveMessage(reqID string, msg *telebot.Message) {
	tg.summariesMu.Lock()
	defer tg.summariesMu.Unlock()
	tg.messages[reqID] = msg
}

func (tg *Bot) getMessage(reqID string) *telebot.Message {
	tg.summariesMu.RLock()
	defer tg.summariesMu.RUnlock()
	return tg.messages[reqID]
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
