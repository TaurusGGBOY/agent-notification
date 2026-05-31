//go:build darwin

package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDarwinNotificationRequestUsesAgentSubtitleAndEventSound(t *testing.T) {
	req := newDarwinNotificationRequest("start", "开始通知", "agent-notification 已启动", "codex", "AgentNotify")

	if req.Title != "开始通知" {
		t.Fatalf("Title = %q", req.Title)
	}
	if req.Body != "agent-notification 已启动" {
		t.Fatalf("Body = %q", req.Body)
	}
	if req.Subtitle != "codex" {
		t.Fatalf("Subtitle = %q", req.Subtitle)
	}
	if req.Sound != "Hero" {
		t.Fatalf("Sound = %q", req.Sound)
	}
}

func TestDarwinNotificationRequestFallsBackToAppNameSubtitle(t *testing.T) {
	req := newDarwinNotificationRequest("stop", "完成通知", "agent-notification 已停止", "  ", "AgentNotify")

	if req.Subtitle != "AgentNotify" {
		t.Fatalf("Subtitle = %q", req.Subtitle)
	}
	if req.Sound != "Glass" {
		t.Fatalf("Sound = %q", req.Sound)
	}
}

func TestDarwinNotificationBundleIdentifierMatchesTauriConfig(t *testing.T) {
	data, err := os.ReadFile("../tauri-app/src-tauri/tauri.conf.json")
	if err != nil {
		t.Fatal(err)
	}

	var config struct {
		Identifier string `json:"identifier"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}

	if darwinNotificationBundleIdentifier != config.Identifier {
		t.Fatalf("darwinNotificationBundleIdentifier = %q, tauri identifier = %q", darwinNotificationBundleIdentifier, config.Identifier)
	}
}
