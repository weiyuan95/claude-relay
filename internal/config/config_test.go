package config

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Port != 7654 {
		t.Errorf("expected port 7654, got %d", cfg.Port)
	}
	if cfg.DefaultMode != "local" {
		t.Errorf("expected default mode 'local', got %s", cfg.DefaultMode)
	}
	if cfg.TelegramModeTimeoutSeconds != 300 {
		t.Errorf("expected timeout 300, got %d", cfg.TelegramModeTimeoutSeconds)
	}
}

func TestLoadFromValidFile(t *testing.T) {
	cfgPath, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath error: %v", err)
	}

	content := Config{
		TelegramBotToken:           "test-token",
		AllowedTelegramUserID:      123456,
		Port:                       8888,
		DefaultMode:                "telegram",
		TelegramModeTimeoutSeconds: 60,
		NotifyInLocalMode:          false,
	}
	data, _ := json.Marshal(content)
	os.WriteFile(cfgPath, data, 0600)
	defer os.Remove(cfgPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TelegramBotToken != "test-token" {
		t.Errorf("expected token 'test-token', got %s", cfg.TelegramBotToken)
	}
	if cfg.Port != 8888 {
		t.Errorf("expected port 8888, got %d", cfg.Port)
	}
	if cfg.AllowedTelegramUserID != 123456 {
		t.Errorf("expected user ID 123456, got %d", cfg.AllowedTelegramUserID)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfgPath, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath error: %v", err)
	}
	os.Remove(cfgPath)

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoadMissingToken(t *testing.T) {
	cfgPath, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath error: %v", err)
	}

	content := Config{AllowedTelegramUserID: 123456}
	data, _ := json.Marshal(content)
	os.WriteFile(cfgPath, data, 0600)
	defer os.Remove(cfgPath)

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for missing telegram_bot_token")
	}
}
