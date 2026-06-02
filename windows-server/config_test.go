package main

import "testing"

func TestConfigNormalizeUnsupportedStyle(t *testing.T) {
	cfg := &Config{
		NotificationStyle: "unknown",
		EnabledEvents:     []string{"start", "stop"},
	}
	cfg.Normalize()
	if cfg.NotificationStyle != "clean" {
		t.Fatalf("NotificationStyle = %q", cfg.NotificationStyle)
	}
}

func TestConfigNormalizeFiltersEvents(t *testing.T) {
	cfg := &Config{
		NotificationStyle: "compact",
		EnabledEvents:     []string{"start", "bad", "stop", "stop"},
	}
	cfg.Normalize()
	want := []string{"start", "stop"}
	if len(cfg.EnabledEvents) != len(want) {
		t.Fatalf("EnabledEvents = %#v", cfg.EnabledEvents)
	}
	for i := range want {
		if cfg.EnabledEvents[i] != want[i] {
			t.Fatalf("EnabledEvents = %#v", cfg.EnabledEvents)
		}
	}
}

func TestConfigNormalizeLanguage(t *testing.T) {
	cfg := &Config{
		NotificationStyle: "clean",
		EnabledEvents:     []string{"start", "stop"},
		Language:          "fr",
	}
	cfg.Normalize()
	if cfg.Language != "zh" {
		t.Fatalf("Language = %q, want zh", cfg.Language)
	}

	cfg.Language = " EN "
	cfg.Normalize()
	if cfg.Language != "en" {
		t.Fatalf("Language = %q, want en", cfg.Language)
	}
}
