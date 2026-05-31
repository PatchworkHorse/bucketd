package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("BUCKETD_ENV", "test")
	copyConfigToWorkingDir(t, "test")

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

func copyConfigToWorkingDir(t *testing.T, env string) {
	t.Helper()

	src := filepath.Join("..", "config", "config."+env+".yaml")
	dst := "config." + env + ".yaml"

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("failed to read source config %s: %v", src, err)
	}

	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("failed to write test config %s: %v", dst, err)
	}

	t.Cleanup(func() {
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			t.Fatalf("failed to remove test config %s: %v", dst, err)
		}
	})
}
