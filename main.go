package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/weiyuanlee/claude-code-relay/internal/config"
	"github.com/weiyuanlee/claude-code-relay/internal/hook"
	"github.com/weiyuanlee/claude-code-relay/internal/model"
	"github.com/weiyuanlee/claude-code-relay/internal/server"
	"github.com/weiyuanlee/claude-code-relay/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	var initialMode model.Mode
	if cfg.DefaultMode == "telegram" {
		initialMode = model.ModeTelegram
	} else {
		initialMode = model.ModeLocal
	}

	// Create handler with nil notifier first (wired after bot creation)
	handler := hook.NewHandler(
		initialMode,
		time.Duration(cfg.TelegramModeTimeoutSeconds)*time.Second,
		cfg.NotifyInLocalMode,
		nil,
	)

	// Create bot with real handler (no nil handler issue)
	bot, err := telegram.NewBot(
		cfg.TelegramBotToken,
		cfg.AllowedTelegramUserID,
		handler,
		fmt.Sprintf("http://127.0.0.1:%d", cfg.Port),
	)
	if err != nil {
		log.Fatalf("Telegram bot error: %v", err)
	}

	// Wire bot as the handler's notifier
	handler.SetNotifier(bot)

	// Create and start HTTP server with graceful shutdown support
	srv := server.New(handler, cfg.Port)
	httpServer := &http.Server{
		Addr:    srv.Addr(),
		Handler: srv.Handler(),
	}

	go func() {
		log.Printf("HTTP server listening on %s", srv.Addr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	go bot.Start()

	log.Printf("Claude Code Relay running (mode: %s, timeout: %ds)", initialMode, cfg.TelegramModeTimeoutSeconds)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	bot.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
}
