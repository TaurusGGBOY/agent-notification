//go:build windows
// +build windows

package main

import (
	"errors"
	"strings"
	"testing"
)

func withFakeToastDeps(t *testing.T) (*string, *string) {
	t.Helper()

	oldLogoPath := toastLogoPathFunc
	oldRenderLogo := renderToastLogoFn
	oldSend := sendToastPowerShell
	t.Cleanup(func() {
		toastLogoPathFunc = oldLogoPath
		renderToastLogoFn = oldRenderLogo
		sendToastPowerShell = oldSend
	})

	var sentAppID string
	var sentXML string
	toastLogoPathFunc = func() (string, error) {
		return `C:\Temp\agent-logo.png`, nil
	}
	renderToastLogoFn = func(path string) error {
		return nil
	}
	sendToastPowerShell = func(appID, xml string) error {
		sentAppID = appID
		sentXML = xml
		return nil
	}
	return &sentAppID, &sentXML
}

func TestWindowsToastNotifierNotifyUsesCleanStyle(t *testing.T) {
	appID, xml := withFakeToastDeps(t)
	notifier := NewToastNotifier("AgentNotify")

	if err := notifier.Notify("Title", "Message"); err != nil {
		t.Fatalf("Notify failed: %v", err)
	}
	if *appID != "AgentNotify" {
		t.Fatalf("appID = %q, want AgentNotify", *appID)
	}
	if !strings.Contains(*xml, `>Title</text>`) {
		t.Fatalf("clean toast XML missing title: %s", *xml)
	}
}

func TestWindowsToastNotifierNotifyWithStyleSendsFormattedXML(t *testing.T) {
	appID, xml := withFakeToastDeps(t)
	notifier := NewToastNotifier("AgentNotify")

	if err := notifier.NotifyWithStyle("agent-badge", "start", "Agent Started", "message", "codex"); err != nil {
		t.Fatalf("NotifyWithStyle failed: %v", err)
	}
	if *appID != "AgentNotify" {
		t.Fatalf("appID = %q, want AgentNotify", *appID)
	}
	if !strings.Contains(*xml, `message`) {
		t.Fatalf("toast XML missing message: %s", *xml)
	}
	if !strings.Contains(*xml, `placement="appLogoOverride"`) {
		t.Fatalf("toast XML missing app logo: %s", *xml)
	}
}

func TestWindowsToastNotifierUsesSingleCleanStyleWithLogo(t *testing.T) {
	_, xml := withFakeToastDeps(t)
	notifier := NewToastNotifier("AgentNotify")

	if err := notifier.NotifyWithStyle("custom-card", "stop", "Agent Stopped", "done", "claude"); err != nil {
		t.Fatalf("NotifyWithStyle failed: %v", err)
	}
	if strings.Contains(*xml, `placement="hero"`) {
		t.Fatalf("toast XML should not use hero image: %s", *xml)
	}
	if !strings.Contains(*xml, `placement="appLogoOverride"`) || !strings.Contains(*xml, `agent-logo.png`) {
		t.Fatalf("toast XML missing app logo image: %s", *xml)
	}
}

func TestWindowsToastNotifierIgnoresLogoPathError(t *testing.T) {
	withFakeToastDeps(t)
	wantErr := errors.New("path failed")
	toastLogoPathFunc = func() (string, error) {
		return "", wantErr
	}

	err := NewToastNotifier("AgentNotify").NotifyWithStyle("custom-card", "start", "title", "message", "agent")
	if err != nil {
		t.Fatalf("logo path error should not block notification: %v", err)
	}
}

func TestWindowsToastNotifierIgnoresLogoRenderError(t *testing.T) {
	withFakeToastDeps(t)
	wantErr := errors.New("render failed")
	renderToastLogoFn = func(path string) error {
		return wantErr
	}

	err := NewToastNotifier("AgentNotify").NotifyWithStyle("custom-card", "start", "title", "message", "agent")
	if err != nil {
		t.Fatalf("logo render error should not block notification: %v", err)
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

func TestNewHiddenPowerShellCommandSuppressesWindow(t *testing.T) {
	cmd := newHiddenPowerShellCommand("Write-Output ok")
	if cmd.SysProcAttr == nil {
		t.Fatal("PowerShell command should set SysProcAttr")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("PowerShell command should hide its window")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatalf("PowerShell command CreationFlags = %#x, want CREATE_NO_WINDOW", cmd.SysProcAttr.CreationFlags)
	}
}

func TestEscapePowerShellSingleQuoted(t *testing.T) {
	if got := escapePowerShellSingleQuoted("Agent's App"); got != "Agent''s App" {
		t.Fatalf("escaped string = %q, want Agent''s App", got)
	}
}
