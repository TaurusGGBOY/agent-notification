package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/grandcat/zeroconf"
)

func discoveryTXT() []string {
	return []string{
		"version=" + version,
		"events=" + strings.Join(supportedEvents(), ","),
		"styles=" + strings.Join(supportedStyles(), ","),
		"path=/notify",
		"settings=/settings",
	}
}

func StartMDNSAdvertisement(ctx context.Context, port uint16) error {
	host := localHostname()
	instance := "Agent Notify " + host

	server, err := zeroconf.Register(instance, mdnsServiceType, "local.", int(port), discoveryTXT(), nil)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		server.Shutdown()
	}()

	log.Printf("mDNS advertising %s as %q on port %d", mdnsServiceType, instance, port)
	return nil
}

type BroadcastController struct {
	mu      sync.Mutex
	port    uint16
	enabled bool
	cancel  context.CancelFunc
}

func NewBroadcastController(port uint16) *BroadcastController {
	return &BroadcastController{port: port}
}

func (c *BroadcastController) Enabled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enabled
}

func (c *BroadcastController) SetEnabled(enabled bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.enabled == enabled {
		return nil
	}

	if !enabled {
		if c.cancel != nil {
			c.cancel()
			c.cancel = nil
		}
		c.enabled = false
		log.Println("mDNS advertising disabled")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := StartMDNSAdvertisement(ctx, c.port); err != nil {
		cancel()
		return err
	}
	c.cancel = cancel
	c.enabled = true
	return nil
}

type BroadcastHandler struct {
	controller *BroadcastController
}

func NewBroadcastHandler(controller *BroadcastController) *BroadcastHandler {
	return &BroadcastHandler{controller: controller}
}

func (h *BroadcastHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(map[string]bool{"enabled": h.controller.Enabled()})
	case http.MethodPost:
		var payload struct {
			Enabled *bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		if payload.Enabled == nil {
			http.Error(w, "enabled is required", http.StatusBadRequest)
			return
		}
		if err := h.controller.SetEnabled(*payload.Enabled); err != nil {
			log.Printf("Failed to update mDNS broadcast: %v", err)
			http.Error(w, "Failed to update broadcast", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"enabled": h.controller.Enabled()})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
