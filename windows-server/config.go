package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	configPathEnv = "APPDATA"
	appName       = "AgentNotify"
	configFile    = "config.json"
)

type Config struct {
	NotificationStyle string            `json:"notificationStyle"`
	EnabledEvents     []string          `json:"enabledEvents"`
	FutureOverrides   map[string]string `json:"futureOverrides"`
}

func LoadConfig() (*Config, error) {
	configDir := os.Getenv(configPathEnv)
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}

	configPath := filepath.Join(configDir, appName, configFile)

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

func (c *Config) Normalize() {
	if !IsSupportedStyle(c.NotificationStyle) {
		c.NotificationStyle = "clean"
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
