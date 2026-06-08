package redisx

import (
	"crypto/tls"
	"time"
)

const (
	defaultAddr         = "127.0.0.1:6379"
	defaultDialTimeout  = 5 * time.Second
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second
)

// Mode 表示 Redis 部署模式。
type Mode string

const (
	// ModeStandalone 表示单机 Redis。
	ModeStandalone Mode = "standalone"
	// ModeSentinel 表示 Redis Sentinel。
	ModeSentinel Mode = "sentinel"
	// ModeCluster 表示 Redis Cluster。
	ModeCluster Mode = "cluster"
)

type options struct {
	mode Mode

	addr  string
	addrs []string

	username string
	password string
	db       int

	masterName       string
	sentinelUsername string
	sentinelPassword string

	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration

	poolSize     int
	minIdleConns int

	tlsConfig *tls.Config
	ping      bool
}

// Option 配置 Redis client。
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(options *options) { f(options) }

// WithAddr 设置单机 Redis 地址。
func WithAddr(addr string) Option {
	return optionFunc(func(options *options) {
		options.addr = addr
	})
}

// WithSentinel 使用 Redis Sentinel。
func WithSentinel(masterName string, addrs ...string) Option {
	return optionFunc(func(options *options) {
		options.mode = ModeSentinel
		options.masterName = masterName
		options.addrs = append([]string(nil), addrs...)
	})
}

// WithSentinelAuth 设置 Sentinel 自身的认证信息。
func WithSentinelAuth(username, password string) Option {
	return optionFunc(func(options *options) {
		options.sentinelUsername = username
		options.sentinelPassword = password
	})
}

// WithCluster 使用 Redis Cluster。
func WithCluster(addrs ...string) Option {
	return optionFunc(func(options *options) {
		options.mode = ModeCluster
		options.addrs = append([]string(nil), addrs...)
	})
}

// WithUsername 设置用户名。
func WithUsername(username string) Option {
	return optionFunc(func(options *options) {
		options.username = username
	})
}

// WithPassword 设置密码。
func WithPassword(password string) Option {
	return optionFunc(func(options *options) {
		options.password = password
	})
}

// WithDB 设置 Redis DB。
func WithDB(db int) Option {
	return optionFunc(func(options *options) {
		options.db = db
	})
}

// WithTimeouts 设置连接、读、写超时。
func WithTimeouts(dial, read, write time.Duration) Option {
	return optionFunc(func(options *options) {
		options.dialTimeout = dial
		options.readTimeout = read
		options.writeTimeout = write
	})
}

// WithPool 设置连接池大小和最小空闲连接数。
func WithPool(size, minIdle int) Option {
	return optionFunc(func(options *options) {
		options.poolSize = size
		options.minIdleConns = minIdle
	})
}

// WithTLSConfig 设置 TLS 配置。
func WithTLSConfig(cfg *tls.Config) Option {
	return optionFunc(func(options *options) {
		options.tlsConfig = cfg
	})
}

// WithPing 控制 New 是否在返回前执行 PING。
func WithPing(enabled bool) Option {
	return optionFunc(func(options *options) {
		options.ping = enabled
	})
}

func defaultOptions() options {
	return options{
		mode:         ModeStandalone,
		addr:         defaultAddr,
		dialTimeout:  defaultDialTimeout,
		readTimeout:  defaultReadTimeout,
		writeTimeout: defaultWriteTimeout,
	}
}
