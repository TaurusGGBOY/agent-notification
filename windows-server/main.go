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

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
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

	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.HealthHandler)
	mux.HandleFunc("/manifest", server.ManifestHandler)
	mux.HandleFunc("/notify", server.NotifyHandler)
	mux.Handle("/settings", NewSettingsHandler())
	mux.Handle("/config", &ConfigHandler{})

	go func() {
		log.Printf("HTTP server listening on %s", httpAddr)
		if err := http.ListenAndServe(httpAddr, withCORS(mux)); err != nil {
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
