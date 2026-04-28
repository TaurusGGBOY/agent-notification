package main

import (
	"context"
	"log"
	"strings"

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