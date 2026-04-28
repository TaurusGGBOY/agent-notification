package main

import (
	"log"
	"net/http"
)

const (
	httpAddr = "0.0.0.0:17891"
)

func main() {
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

	go StartUDPDiscovery()

	log.Println("Server started successfully")
	select {}
}
