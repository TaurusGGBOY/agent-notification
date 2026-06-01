package main

import (
	"context"
	"log"

	"github.com/grandcat/zeroconf"
)

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
