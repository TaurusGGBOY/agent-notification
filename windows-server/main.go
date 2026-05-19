package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
)

func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func main() {
	httpAddr := envOrDefault("AGENT_NOTIFY_HTTP_ADDR", "0.0.0.0:17891")

	log.Println("Starting Agent Notify Server...")

	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("Warning: Failed to load config: %v, using defaults", err)
		cfg = DefaultConfig()
	}

	log.Printf("Config loaded: notificationStyle=%s, enabledEvents=%v",
		cfg.NotificationStyle, cfg.EnabledEvents)

	server := NewServer(cfg)

	http.HandleFunc("/health", server.HealthHandler)
	http.HandleFunc("/manifest", server.ManifestHandler)
	http.HandleFunc("/notify", server.NotifyHandler)
	http.Handle("/settings", NewSettingsHandler())
	http.Handle("/config", &ConfigHandler{})

	go func() {
		log.Printf("HTTP server listening on %s", httpAddr)
		if err := http.ListenAndServe(httpAddr, nil); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := StartMDNSAdvertisement(ctx, 17891); err != nil {
		log.Printf("Warning: mDNS advertisement failed: %v", err)
	}

	log.Println("Server started successfully")
	select {}
}
