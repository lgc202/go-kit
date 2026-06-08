package redisx

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewDefaultsToStandaloneClient(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	standalone, ok := client.(*redis.Client)
	if !ok {
		t.Fatalf("New() = %T, want *redis.Client", client)
	}
	opts := standalone.Options()
	if opts.Addr != defaultAddr {
		t.Errorf("New().Options().Addr = %q, want %q", opts.Addr, defaultAddr)
	}
	if opts.DialTimeout != defaultDialTimeout {
		t.Errorf("New().Options().DialTimeout = %v, want %v", opts.DialTimeout, defaultDialTimeout)
	}
	if opts.ReadTimeout != defaultReadTimeout {
		t.Errorf("New().Options().ReadTimeout = %v, want %v", opts.ReadTimeout, defaultReadTimeout)
	}
	if opts.WriteTimeout != defaultWriteTimeout {
		t.Errorf("New().Options().WriteTimeout = %v, want %v", opts.WriteTimeout, defaultWriteTimeout)
	}
}

func TestNewStandaloneAppliesOptions(t *testing.T) {
	tlsConfig := &tls.Config{ServerName: "redis.internal"}
	client, err := New(
		WithAddr("redis.internal:6380"),
		WithUsername("alice"),
		WithPassword("secret"),
		WithDB(2),
		WithTimeouts(2*time.Second, 4*time.Second, 5*time.Second),
		WithPool(20, 3),
		WithTLSConfig(tlsConfig),
	)
	if err != nil {
		t.Fatalf("New(standalone options) error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	opts := client.(*redis.Client).Options()
	if opts.Addr != "redis.internal:6380" {
		t.Errorf("New(standalone options).Options().Addr = %q, want redis.internal:6380", opts.Addr)
	}
	if opts.Username != "alice" {
		t.Errorf("New(standalone options).Options().Username = %q, want alice", opts.Username)
	}
	if opts.Password != "secret" {
		t.Errorf("New(standalone options).Options().Password = %q, want secret", opts.Password)
	}
	if opts.DB != 2 {
		t.Errorf("New(standalone options).Options().DB = %d, want 2", opts.DB)
	}
	if opts.PoolSize != 20 {
		t.Errorf("New(standalone options).Options().PoolSize = %d, want 20", opts.PoolSize)
	}
	if opts.MinIdleConns != 3 {
		t.Errorf("New(standalone options).Options().MinIdleConns = %d, want 3", opts.MinIdleConns)
	}
	if opts.TLSConfig != tlsConfig {
		t.Errorf("New(standalone options).Options().TLSConfig = %p, want %p", opts.TLSConfig, tlsConfig)
	}
}

func TestNewClusterClient(t *testing.T) {
	client, err := New(WithCluster("10.0.0.1:6379", "10.0.0.2:6379"))
	if err != nil {
		t.Fatalf("New(cluster options) error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	cluster, ok := client.(*redis.ClusterClient)
	if !ok {
		t.Fatalf("New(cluster options) = %T, want *redis.ClusterClient", client)
	}
	opts := cluster.Options()
	if len(opts.Addrs) != 2 || opts.Addrs[0] != "10.0.0.1:6379" || opts.Addrs[1] != "10.0.0.2:6379" {
		t.Errorf("New(cluster options).Options().Addrs = %v, want [10.0.0.1:6379 10.0.0.2:6379]", opts.Addrs)
	}
}

func TestNewSentinelRequiresMasterName(t *testing.T) {
	if _, err := New(WithSentinel("", "10.0.0.1:26379")); err == nil {
		t.Fatalf("New(sentinel without master name) error = nil, want error")
	}
}

func TestWithSentinelAuthAppliesOptions(t *testing.T) {
	options := defaultOptions()
	WithSentinelAuth("sentinel-user", "sentinel-secret").apply(&options)

	if options.sentinelUsername != "sentinel-user" {
		t.Fatalf("sentinelUsername = %q, want sentinel-user", options.sentinelUsername)
	}
	if options.sentinelPassword != "sentinel-secret" {
		t.Fatalf("sentinelPassword = %q, want sentinel-secret", options.sentinelPassword)
	}
}

func TestNewRejectsInvalidMode(t *testing.T) {
	if _, err := New(testMode("proxy")); err == nil {
		t.Fatalf("New(invalid mode) error = nil, want error")
	}
}

func TestNewRejectsNilOption(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatalf("New(nil) error = nil, want error")
	}
}

func TestNewWithContextUsesContextForPing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewWithContext(ctx, WithPing(true))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("NewWithContext(canceled ctx, WithPing(true)) error = %v, want context.Canceled", err)
	}
}

type testMode Mode

func (m testMode) apply(options *options) {
	options.mode = Mode(m)
}
