package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	version         = "1.0.0"
	mdnsServiceType = "_agent-notify._tcp"
)

func supportedEvents() []string {
	return []string{"start", "stop"}
}

func supportedStyles() []string {
	return []string{"clean", "status-color", "agent-badge", "compact", "custom-card"}
}

func localHostname() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "unknown"
	}
	return host
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type ManifestResponse struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	URL             string   `json:"url"`
	Hostname        string   `json:"hostname"`
	Protocol        string   `json:"protocol"`
	ServiceType     string   `json:"serviceType"`
	Description     string   `json:"description"`
	SupportedEvents []string `json:"supportedEvents"`
	SupportedStyles []string `json:"supportedStyles"`
}

type NotifyPayload struct {
	Agent         string                 `json:"agent"`
	Event         string                 `json:"event"`
	Project       string                 `json:"project"`
	Cwd           string                 `json:"cwd"`
	Message       string                 `json:"message"`
	Timestamp    string                 `json:"timestamp"`
	SourcePayload map[string]interface{} `json:"sourcePayload"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Server struct {
	config   *Config
	notifier Notifier
}

func NewServer(cfg *Config) *Server {
	return &Server{
		config:   cfg,
		notifier: NewToastNotifier("AgentNotify"),
	}
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := HealthResponse{
		Status:  "ok",
		Version: version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) ManifestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := ManifestResponse{
		Name:            "Agent Notify Server",
		Version:         version,
		URL:             "http://" + r.Host,
		Hostname:        localHostname(),
		Protocol:        "mdns-dns-sd",
		ServiceType:     mdnsServiceType + ".local.",
		Description:     "Windows notification server for agent start/stop events",
		SupportedEvents: supportedEvents(),
		SupportedStyles: supportedStyles(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) NotifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload NotifyPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON payload"})
		return
	}

	event := strings.ToLower(strings.TrimSpace(payload.Event))
	if event != "start" && event != "stop" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "event must be 'start' or 'stop'"})
		return
	}

	if !s.config.IsEventEnabled(event) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	title := formatTitle(payload.Agent, event)
	message := formatMessage(payload)

	if err := s.notifier.NotifyWithStyle(
		s.config.NotificationStyle,
		event,
		title,
		message,
		payload.Agent,
	); err != nil {
		log.Printf("Toast notification failed: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func formatTitle(agent, event string) string {
	eventEmoji := map[string]string{
		"start": "🚀",
		"stop":  "⏹️",
	}

	emoji := eventEmoji[event]
	if emoji == "" {
		emoji = "📢"
	}

	return emoji + " Agent " + strings.Title(event) + ": " + agent
}

func formatMessage(payload NotifyPayload) string {
	parts := []string{}

	if payload.Project != "" {
		parts = append(parts, "Project: "+payload.Project)
	}

	if payload.Cwd != "" {
		parts = append(parts, "CWD: "+payload.Cwd)
	}

	if payload.Message != "" {
		parts = append(parts, payload.Message)
	}

	if payload.Timestamp != "" {
		if _, err := time.Parse(time.RFC3339, payload.Timestamp); err == nil {
			if t, err := time.Parse(time.RFC3339, payload.Timestamp); err == nil {
				parts = append(parts, "at "+t.Format("15:04:05"))
			}
		}
	}

	if len(parts) == 0 {
		return "Agent event occurred"
	}

	return strings.Join(parts, " | ")
}
