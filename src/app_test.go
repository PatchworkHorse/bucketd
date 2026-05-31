package main

import (
	"os"
	"slices"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("BUCKETD_ENV", "test")
	t.Setenv("BUCKETD_CORE__MODE", "dns")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir("../config"); err != nil {
		t.Fatalf("failed to change to config directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("loadConfig panicked: %v", r)
		}
	}()

	coreConfig, httpConfig, dnsConfig, redisConfig, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig returned an error: %v", err)
	}

	if coreConfig == nil || httpConfig == nil || dnsConfig == nil || redisConfig == nil {
		t.Fatal("loadConfig returned nil config pointer")
	}

	// Test HTTP hostnames
	if !slices.Contains(httpConfig.Hostnames, "object.patchwork.horse") {
		t.Fatal("HTTP hostnames does not contain 'object.patchwork.horse'")
	}

	// Test DNS FQDNs
	if !slices.Contains(dnsConfig.FQDNs, "object.patchwork.horse.") {
		t.Fatal("DNS FQDNs does not contain 'object.patchwork.horse.'")
	}
}
