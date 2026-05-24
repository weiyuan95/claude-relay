package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	TelegramBotToken           string `json:"telegram_bot_token"`
	AllowedTelegramUserID      int64  `json:"allowed_telegram_user_id"`
	Port                       int    `json:"port"`
	DefaultMode                string `json:"default_mode"`
	TelegramModeTimeoutSeconds int    `json:"telegram_mode_timeout_seconds"`
	NotifyInLocalMode          bool   `json:"notify_in_local_mode"`
}

func DefaultConfig() Config {
	return Config{
		Port:                       7654,
		DefaultMode:                "local",
		TelegramModeTimeoutSeconds: 300,
		NotifyInLocalMode:          true,
	}
}

func ConfigPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), ".claude-relay.json"), nil
}

func Load() (Config, error) {
	cfg := DefaultConfig()

	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, fmt.Errorf("config file not found at %s: create it with telegram_bot_token and allowed_telegram_user_id", path)
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	if cfg.TelegramBotToken == "" {
		return cfg, fmt.Errorf("telegram_bot_token is required in %s", path)
	}

	if cfg.AllowedTelegramUserID == 0 {
		return cfg, fmt.Errorf("allowed_telegram_user_id is required in %s", path)
	}

	return cfg, nil
}
