package main

import (
	"fmt"
	"strings"

	"github.com/containerd/log"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"patchwork.horse/bucketd/bucketDns"
	"patchwork.horse/bucketd/bucketHttp"
	"patchwork.horse/bucketd/config"
)

func main() {
	k := koanf.New(".")

	mapper := func(s string) string {
		s = strings.TrimPrefix(s, "BUCKETD_")
		s = strings.ReplaceAll(s, "__", ".")
		return strings.ToLower(s)
	}

	if err := k.Load(file.Provider("config.prod.yaml"), yaml.Parser()); err != nil {
		fmt.Printf("Failed to load config file: %v\n", err)
	}
	if err := k.Load(env.Provider("BUCKETD_", ".", mapper), nil); err != nil {
		fmt.Printf("Failed to load environment variables: %v\n", err)
	}

	var coreConfig config.CoreConfig
	var httpConfig config.HttpConfig
	var dnsConfig config.DnsConfig
	var redisConfig config.RedisConfig

	if e := k.Unmarshal(config.CoreKey, &coreConfig); e != nil {
		log.L.WithError(e).Warnf("Failed to load config section %s", config.CoreKey)
	}
	if e := k.Unmarshal(config.HTTPKey, &httpConfig); e != nil {
		log.L.WithError(e).Warnf("Failed to load config section %s", config.HTTPKey)
	}
	if e := k.Unmarshal(config.DNSKey, &dnsConfig); e != nil {
		log.L.WithError(e).Warnf("Failed to load config section %s", config.DNSKey)
	}
	if e := k.Unmarshal(config.RedisKey, &redisConfig); e != nil {
		log.L.WithError(e).Warnf("Failed to load config section %s", config.RedisKey)
	}

	switch strings.ToLower(coreConfig.Mode) {

	case "http":
		bucketHttp.StartHttpListener(&coreConfig, &httpConfig, &redisConfig)

	case "dns":
		bucketDns.StartDnsListener(&coreConfig, &dnsConfig, &redisConfig)

	default:
		panic("Invalid mode in core config, must be 'http' or 'dns'")
	}
}
