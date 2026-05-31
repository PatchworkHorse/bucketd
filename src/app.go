package main

import (
	"flag"
	"fmt"
	"os"
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

func Run() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s [options]\n", os.Args[0])
		printEnvVarHelp()
	}
	flag.Parse()

	// Ensure required environment variables are set before we bother loading the config
	if err := validateEnvVars(); err != nil {
		panic(err)
	}

	// Load the configs, then start the appropriate listener
	coreConfig, httpConfig, dnsConfig, redisConfig, err := loadConfig()
	if err != nil {
		panic(err)
	}

	switch strings.ToLower(coreConfig.Mode) {

	case "http":
		bucketHttp.StartHttpListener(coreConfig, httpConfig, redisConfig)

	case "dns":
		bucketDns.StartDnsListener(coreConfig, dnsConfig, redisConfig)

	default:
		panic("Invalid mode in core config, must be 'http' or 'dns'")
	}
}

func validateEnvVars() error {
	requiredVars := []string{
		"BUCKETD_ENV",
	}

	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			return fmt.Errorf("environment variable %s is not set", v)
		}
	}
	return nil
}

func loadConfig() (*config.CoreConfig, *config.HttpConfig, *config.DnsConfig, *config.RedisConfig, error) {

	bucked_env := os.Getenv("BUCKETD_ENV")

	mapper := func(s string) string {
		s = strings.TrimPrefix(s, "BUCKETD_")
		s = strings.ReplaceAll(s, "__", ".")
		return strings.ToLower(s)
	}

	k := koanf.New(".")

	if err := k.Load(file.Provider(fmt.Sprintf("config.%s.yaml", bucked_env)), yaml.Parser()); err != nil {
		fmt.Printf("Failed to load config file: %v\n", err)
		return nil, nil, nil, nil, err
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

	return &coreConfig, &httpConfig, &dnsConfig, &redisConfig, nil
}

func printEnvVarHelp() {
	fmt.Print(`
Supported environment variables:

BUCKETD_ENV
  Selects which config file to load (config.<env>.yaml).
  Example: BUCKETD_ENV=test loads config.test.yaml

`)
}
