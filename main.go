package main

import (
	"context"
	"flag"
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

	mode := flag.String("mode", "", "Override default mode: local or telegram (overrides config)")
	timeout := flag.Int("timeout", 0, "Telegram mode timeout in seconds (overrides config)")
	notify := flag.Bool("notify", true, "Send Telegram notifications in local mode")
	flag.Parse()

	if *mode != "" {
		if *mode != "local" && *mode != "telegram" {
			log.Fatalf("Invalid mode: %s (use 'local' or 'telegram')", *mode)
		}
		cfg.DefaultMode = *mode
	}
	if *timeout > 0 {
		cfg.TelegramModeTimeoutSeconds = *timeout
	}
	cfg.NotifyInLocalMode = *notify

	var initialMode model.Mode
	if cfg.DefaultMode == "telegram" {
		initialMode = model.ModeTelegram
	} else {
		initialMode = model.ModeLocal
	}

	handler := hook.NewHandler(
		initialMode,
		time.Duration(cfg.TelegramModeTimeoutSeconds)*time.Second,
		cfg.NotifyInLocalMode,
		nil,
	)

	bot, err := telegram.NewBot(
		cfg.TelegramBotToken,
		cfg.AllowedTelegramUserID,
		handler,
		fmt.Sprintf("http://127.0.0.1:%d", cfg.Port),
	)
	if err != nil {
		log.Fatalf("Telegram bot error: %v", err)
	}

	handler.SetNotifier(bot)

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

	log.Printf("Claude Code Relay running (mode: %s, timeout: %ds, notify: %v)", initialMode, cfg.TelegramModeTimeoutSeconds, cfg.NotifyInLocalMode)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	handler.ShutdownAll()
	bot.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
}
