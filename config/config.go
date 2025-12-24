// Package config provides configuration structs and constructors for bucketd.
package config

import (
	"net"
)

const (
	CoreKey  = "core"
	HTTPKey  = "http"
	DNSKey   = "dns"
	RedisKey = "redis"
)

// CoreConfig represents the global rules for caching values that apply to all providers
type CoreConfig struct {
	// Mode defines what service will run. Valid values are HTTP and DNS
	Mode string
	// MaxKeyLength represents the max key length, in bytes
	MaxKeyLength int // bytes
	// MaxValueLength represents the max value length, in bytes
	MaxValueLength int // bytes
	// MaxTTL represents the max time to live for a given key
	MaxTtl int // seconds,
	// MaxElements is the maximum allowed elements we can store at any given time. Reject new entries until space becomes available
	MaxElements int
}

// DnsConfig provides DNS-provider specific configurations, consumed by BucketDns
type DnsConfig struct {
	// Port defines the port on which to listen for DNS queries
	Port int
	// FQDN to which BucketDns is bound.
	// The service will only provide answers for questions to this FQDN. Don't forget the trailing dot.
	FQDN string
	// A is the IPv4 answer to be provided for A queries to the FQDN
	A net.IP
	// AAAA is the IPv6 answer to be provided for AAAA queries to the FQDN
	AAAA net.IP
}

// HttpConfig provides HTTP-provider specific configurations
type HttpConfig struct {
	// Port represents the port on which to listen for HTTP requests
	Port int
	// Hostname represents the hostname expected for HTTP requests
	Hostname string
}

// RedisConfig provides Redis-specific configurations if using a Redis cache provider backend
type RedisConfig struct {
	// Address is the hostname and port (host:port) for the Redis instance
	Address string
	// Password is the password to use with Redis, may be left blank if none
	Password string
	// Database is used to specifiy which Redis DB to use, use 0 or leave blank to use the default
	Database int
}

// NewCoreConfig creates a new instance of CoreConfig with reasonable defaults
func NewCoreConfig() CoreConfig {
	return CoreConfig{}
}

// NewRedisConfig creates a new instance of RedisConfig with reasonable defaults
func NewRedisConfig() RedisConfig {
	return RedisConfig{
		Address:  "127.0.0.1:6379",
		Database: 0,
	}
}

// NewHttpConfig creates a new instance of HttpConfig with reasonable defaults
func NewHttpConfig() HttpConfig {
	return HttpConfig{
		Port: 8080,
	}
}

// NewDnsConfig creates a new instance of DnsConfig with reasonable defaults
func NewDnsConfig() DnsConfig {
	return DnsConfig{
		Port: 8053,
	}
}
