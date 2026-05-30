package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
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
	return []string{"clean", "status-color", "agent-badge", "compact"}
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

type NotificationHistoryItem struct {
	Time    string `json:"time"`
	Agent   string `json:"agent"`
	Event   string `json:"event"`
	Project string `json:"project"`
	Message string `json:"message"`
}

type HistoryResponse struct {
	Items []NotificationHistoryItem `json:"items"`
}

type NotifyPayload struct {
	Agent         string                 `json:"agent"`
	Event         string                 `json:"event"`
	Project       string                 `json:"project"`
	Cwd           string                 `json:"cwd"`
	Workdir       string                 `json:"workdir"`
	Message       string                 `json:"message"`
	Timestamp     string                 `json:"timestamp"`
	SourcePayload map[string]interface{} `json:"sourcePayload"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Server struct {
	config    *Config
	notifier  Notifier
	historyMu sync.Mutex
	history   []NotificationHistoryItem
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
		URL:             "http://" + lanHostPort(r.Host),
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

func lanHostPort(fallbackHost string) string {
	host, port, err := net.SplitHostPort(fallbackHost)
	if err != nil {
		host = fallbackHost
		port = "17891"
	}
	if ip := firstLANIPv4(); ip != "" {
		return net.JoinHostPort(ip, port)
	}
	return net.JoinHostPort(host, port)
}

func firstLANIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			return ip4.String()
		}
	}
	return ""
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

	s.recordHistory(payload, event)

	cfg := s.currentConfig()
	if !cfg.IsEventEnabled(event) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	title := formatTitle(payload.Agent, event)
	message := formatMessage(payload)

	if err := s.notifier.NotifyWithStyle(
		cfg.NotificationStyle,
		event,
		title,
		message,
		payload.Agent,
	); err != nil {
		log.Printf("Toast notification failed: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) HistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.historyMu.Lock()
	items := append([]NotificationHistoryItem(nil), s.history...)
	s.historyMu.Unlock()
	if items == nil {
		items = []NotificationHistoryItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HistoryResponse{Items: items})
}

func (s *Server) recordHistory(payload NotifyPayload, event string) {
	item := NotificationHistoryItem{
		Time:    time.Now().Format(time.RFC3339),
		Agent:   strings.TrimSpace(payload.Agent),
		Event:   event,
		Project: strings.TrimSpace(payload.Project),
		Message: strings.TrimSpace(payload.Message),
	}
	if item.Agent == "" {
		item.Agent = "unknown"
	}

	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	s.history = append([]NotificationHistoryItem{item}, s.history...)
	if len(s.history) > 3 {
		s.history = s.history[:3]
	}
}

func (s *Server) currentConfig() *Config {
	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("Failed to reload config: %v", err)
		return s.config
	}
	s.config = cfg
	return cfg
}

func formatTitle(agent, event string) string {
	agent = strings.TrimSpace(agent)
	if agent == "" {
		agent = "unknown"
	}
	return eventLabel(event) + " · " + agent
}

func formatMessage(payload NotifyPayload) string {
	parts := []string{}

	workdir := strings.TrimSpace(payload.Cwd)
	if workdir == "" {
		workdir = strings.TrimSpace(payload.Workdir)
	}
	project := strings.TrimSpace(payload.Project)
	if workdir == "" && looksLikePath(project) {
		workdir = project
		project = ""
	}
	if workdir != "" {
		parts = append(parts, "DIR: "+compactDirectoryForDisplay(workdir, 46))
	}

	if project != "" && project != workdir && !strings.EqualFold(project, "unknown") {
		parts = append(parts, "PROJECT: "+project)
	}

	if message := strings.TrimSpace(payload.Message); message != "" {
		parts = append(parts, message)
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

func eventLabel(event string) string {
	switch strings.ToLower(strings.TrimSpace(event)) {
	case "start":
		return "START"
	case "stop":
		return "STOP"
	default:
		return "EVENT"
	}
}

func looksLikePath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, "~/") ||
		strings.Contains(value, `\`) ||
		strings.Contains(value, "/")
}
