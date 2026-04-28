package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

const (
	udpPort        = 17892
	discoveryMagic = "AGENT_NOTIFY_DISCOVER v1"
)

type DiscoveryResponse struct {
	URL              string   `json:"url"`
	Hostname         string   `json:"hostname"`
	SupportedEvents  []string `json:"supportedEvents"`
	SupportedStyles  []string `json:"supportedStyles"`
}

func StartUDPDiscovery() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", udpPort))
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer conn.Close()

	log.Printf("UDP discovery listening on port %d", udpPort)

	buffer := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			log.Printf("UDP read error: %v", err)
			continue
		}

		msg := string(buffer[:n])
		msg = strings.TrimSpace(msg)

		if msg == discoveryMagic {
			log.Printf("Discovery request from %s", clientAddr.String())

			hostname, _ := getHostname()
			resp := DiscoveryResponse{
				URL:             fmt.Sprintf("http://%s:17891", hostname),
				Hostname:        hostname,
				SupportedEvents:  strings.Split(supportedEvents, ","),
				SupportedStyles: strings.Split(supportedStyles, ","),
			}

			respJSON, _ := json.Marshal(resp)
			_, err := conn.WriteToUDP(respJSON, clientAddr)
			if err != nil {
				log.Printf("UDP write error: %v", err)
			}
		}
	}
}

func getHostname() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "unknown", err
	}
	return name, nil
}
