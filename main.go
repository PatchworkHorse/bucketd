package main

import (
	"fmt"
	"strings"

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

	_ = k.Unmarshal(config.CoreKey, &coreConfig)
	_ = k.Unmarshal(config.HTTPKey, &httpConfig)
	_ = k.Unmarshal(config.DNSKey, &dnsConfig)
	_ = k.Unmarshal(config.RedisKey, &redisConfig)

	fmt.Printf("DEBUG: coreConfig.Mode = '%s'\n", coreConfig.Mode)

	switch strings.ToLower(coreConfig.Mode) {

	case "http":
		httpConfig := config.NewHttpConfig()
		bucketHttp.StartHttpListener(&coreConfig, &httpConfig, &redisConfig)

	case "dns":
		dnsConfig := config.NewDnsConfig()
		bucketDns.StartDnsListener(&coreConfig, &dnsConfig, &redisConfig)

	default:
		panic("Invalid mode in core config, must be 'http' or 'dns'")
	}
}
