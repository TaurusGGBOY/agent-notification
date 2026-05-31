package main

import (
	"strings"
	"testing"
)

func TestDiscoveryTXTContainsContractFields(t *testing.T) {
	txt := strings.Join(discoveryTXT(), "\n")
	for _, want := range []string{
		"version=",
		"events=start,stop",
		"styles=clean",
		"path=/notify",
		"settings=/settings",
	} {
		if !strings.Contains(txt, want) {
			t.Fatalf("discovery TXT missing %q in %q", want, txt)
		}
	}
}

func TestMDNSServiceType(t *testing.T) {
	if mdnsServiceType != "_agent-notify._tcp" {
		t.Fatalf("mdnsServiceType = %q", mdnsServiceType)
	}
}
