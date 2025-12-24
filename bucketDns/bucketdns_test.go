package bucketDns

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"patchwork.horse/bucketd/config"
)

func startRedisTestContainer(ctx context.Context) (client *redis.Client, cleanup func(), err error) {

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},

		// Wait for Redis to listen
		WaitingFor: wait.ForLog("Ready to accept connections"),
	}

	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, nil, err
	}

	endpoint, err := redisC.Endpoint(ctx, "")

	if err != nil {
		return nil, nil, err
	}

	client = redis.NewClient(&redis.Options{
		Addr: endpoint,
	})

	cleanup = func() {
		if err := redisC.Terminate(ctx); err != nil {
			fmt.Printf("Failed to terminate container %s", err)
		}
	}

	return client, cleanup, nil

}

func TestContainerInit(t *testing.T) {
	ctx := context.Background()
	_, cleanup, err := startRedisTestContainer(ctx)

	if err != nil {
		t.Fatalf("Test container failed to start: %v", err)
	}

	t.Log("✅ - Successfully started Redis test container")

	cleanup()

}

func TestHandleTx(t *testing.T) {

	// Arrange
	ctx := context.Background()

	rClient, cleanupTestContainer, err := startRedisTestContainer(ctx)

	// Seed some data
	rClient.Set(ctx, "test", "value", time.Duration(20)*time.Second)

	if err != nil {
		t.Fatalf("Test container failed to start: %v", err)
	}

	// Start our DNS listener
	go func() {
		cConfig := config.NewCoreConfig()
		dConfig := config.NewDnsConfig()
		dConfig.FQDN = "object.patchwork.horse."
		rConfig := config.RedisConfig{
			Address: rClient.Options().Addr,
		}
		StartDnsListener(&cConfig, &dConfig, &rConfig)
	}()

	// Lazy wait to wait for DNS server to be up
	time.Sleep(1 * time.Second)

	dnsClient := dns.Client{}
	m := dns.Msg{}
	m.SetQuestion("test.object.patchwork.horse.", dns.TypeTXT)

	// Act
	response, _, err := dnsClient.Exchange(&m, "127.0.0.1:8053")

	// Assert
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if response.Rcode != dns.RcodeSuccess {
		t.Fatalf("Expected DNS RcodeSuccess, got %s", dns.RcodeToString[response.Rcode])
	}

	t.Logf("DNS Response: %+v", response)

	// Cleanup
	cleanupTestContainer()
}
