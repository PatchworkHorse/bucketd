package main

import "testing"

func TestLoadConfig(t *testing.T) {
	t.Setenv("BUCKETD_ENV", "test")

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

	t.Log("Hello world!")
}
