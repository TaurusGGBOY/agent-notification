//go:build windows
// +build windows

package main

import (
	"errors"
	"strings"
	"testing"
)

func withFakeToastDeps(t *testing.T) (*string, *string, *ToastCard) {
	t.Helper()

	oldPath := toastCardPathFunc
	oldRender := renderToastCardFn
	oldSend := sendToastPowerShell
	t.Cleanup(func() {
		toastCardPathFunc = oldPath
		renderToastCardFn = oldRender
		sendToastPowerShell = oldSend
	})

	var sentAppID string
	var sentXML string
	var renderedCard ToastCard
	toastCardPathFunc = func() (string, error) {
		return `C:\Temp\agent-card.png`, nil
	}
	renderToastCardFn = func(path string, card ToastCard) error {
		renderedCard = card
		return nil
	}
	sendToastPowerShell = func(appID, xml string) error {
		sentAppID = appID
		sentXML = xml
		return nil
	}
	return &sentAppID, &sentXML, &renderedCard
}

func TestWindowsToastNotifierNotifyUsesCleanStyle(t *testing.T) {
	appID, xml, _ := withFakeToastDeps(t)
	notifier := NewToastNotifier("AgentNotify")

	if err := notifier.Notify("Title", "Message"); err != nil {
		t.Fatalf("Notify failed: %v", err)
	}
	if *appID != "AgentNotify" {
		t.Fatalf("appID = %q, want AgentNotify", *appID)
	}
	if !strings.Contains(*xml, "<text>Title</text>") {
		t.Fatalf("clean toast XML missing title: %s", *xml)
	}
}

func TestWindowsToastNotifierNotifyWithStyleSendsFormattedXML(t *testing.T) {
	appID, xml, _ := withFakeToastDeps(t)
	notifier := NewToastNotifier("AgentNotify")

	if err := notifier.NotifyWithStyle("agent-badge", "start", "Agent Started", "message", "codex"); err != nil {
		t.Fatalf("NotifyWithStyle failed: %v", err)
	}
	if *appID != "AgentNotify" {
		t.Fatalf("appID = %q, want AgentNotify", *appID)
	}
	if !strings.Contains(*xml, `placement="attribution">Agent C`) {
		t.Fatalf("agent badge XML missing attribution: %s", *xml)
	}
}

func TestWindowsToastNotifierCustomCardRendersImage(t *testing.T) {
	_, xml, card := withFakeToastDeps(t)
	notifier := NewToastNotifier("AgentNotify")

	if err := notifier.NotifyWithStyle("custom-card", "stop", "Agent Stopped", "done", "claude"); err != nil {
		t.Fatalf("custom card NotifyWithStyle failed: %v", err)
	}
	if card.Event != "stop" || card.Title != "Agent Stopped" || card.Agent != "claude" || card.Message != "done" {
		t.Fatalf("rendered card = %+v", *card)
	}
	if !strings.Contains(*xml, `placement="hero"`) || !strings.Contains(*xml, `agent-card.png`) {
		t.Fatalf("custom card XML missing image: %s", *xml)
	}
}

func TestWindowsToastNotifierReturnsCardPathError(t *testing.T) {
	withFakeToastDeps(t)
	wantErr := errors.New("path failed")
	toastCardPathFunc = func() (string, error) {
		return "", wantErr
	}

	err := NewToastNotifier("AgentNotify").NotifyWithStyle("custom-card", "start", "title", "message", "agent")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestWindowsToastNotifierReturnsRenderError(t *testing.T) {
	withFakeToastDeps(t)
	wantErr := errors.New("render failed")
	renderToastCardFn = func(path string, card ToastCard) error {
		return wantErr
	}

	err := NewToastNotifier("AgentNotify").NotifyWithStyle("custom-card", "start", "title", "message", "agent")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestWindowsToastNotifierReturnsSendError(t *testing.T) {
	withFakeToastDeps(t)
	wantErr := errors.New("send failed")
	sendToastPowerShell = func(appID, xml string) error {
		return wantErr
	}

	err := NewToastNotifier("AgentNotify").NotifyWithStyle("clean", "start", "title", "message", "agent")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestEscapePowerShellSingleQuoted(t *testing.T) {
	if got := escapePowerShellSingleQuoted("Agent's App"); got != "Agent''s App" {
		t.Fatalf("escaped string = %q, want Agent''s App", got)
	}
}
