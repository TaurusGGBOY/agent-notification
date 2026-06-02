package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	configPathEnv = "APPDATA"
	appName       = "AgentNotify"
	configFile    = "config.json"
)

// configDir returns the platform-appropriate config directory for AgentNotify.
func configDir() string {
	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Application Support", appName)
		}
	}
	// Windows: %APPDATA%\AgentNotify
	dir := os.Getenv(configPathEnv)
	if dir == "" {
		dir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	return filepath.Join(dir, appName)
}

type Config struct {
	NotificationStyle string            `json:"notificationStyle"`
	EnabledEvents     []string          `json:"enabledEvents"`
	Language          string            `json:"language"`
	FutureOverrides   map[string]string `json:"futureOverrides"`
}

func LoadConfig() (*Config, error) {
	configPath := filepath.Join(configDir(), configFile)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.Normalize()
	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		NotificationStyle: "clean",
		EnabledEvents:     []string{"start", "stop"},
		Language:          "zh",
		FutureOverrides:   make(map[string]string),
	}
}

func (c *Config) IsEventEnabled(event string) bool {
	for _, e := range c.EnabledEvents {
		if e == event {
			return true
		}
	}
	return false
}

func IsSupportedStyle(style string) bool {
	return style == "clean"
}

func IsSupportedEvent(event string) bool {
	return event == "start" || event == "stop"
}

func IsSupportedLanguage(language string) bool {
	language = strings.ToLower(strings.TrimSpace(language))
	return language == "zh" || language == "en"
}

func (c *Config) Normalize() {
	if !IsSupportedStyle(c.NotificationStyle) {
		c.NotificationStyle = "clean"
	}
	c.Language = strings.ToLower(strings.TrimSpace(c.Language))
	if !IsSupportedLanguage(c.Language) {
		c.Language = "zh"
	}
	if c.EnabledEvents == nil {
		c.EnabledEvents = []string{"start", "stop"}
		return
	}
	filtered := make([]string, 0, len(c.EnabledEvents))
	seen := map[string]bool{}
	for _, event := range c.EnabledEvents {
		if IsSupportedEvent(event) && !seen[event] {
			filtered = append(filtered, event)
			seen[event] = true
		}
	}
	c.EnabledEvents = filtered
	if c.FutureOverrides == nil {
		c.FutureOverrides = map[string]string{}
	}
}
