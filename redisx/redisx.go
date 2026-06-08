package redisx

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// New 根据 opts 创建 Redis client。
func New(opts ...Option) (redis.UniversalClient, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt.apply(&options)
	}

	client, err := newClient(options)
	if err != nil {
		return nil, err
	}
	if !options.ping {
		return client, nil
	}

	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func newClient(options options) (redis.UniversalClient, error) {
	switch options.mode {
	case ModeStandalone:
		return redis.NewClient(&redis.Options{
			Addr:         options.addr,
			Username:     options.username,
			Password:     options.password,
			DB:           options.db,
			DialTimeout:  options.dialTimeout,
			ReadTimeout:  options.readTimeout,
			WriteTimeout: options.writeTimeout,
			PoolSize:     options.poolSize,
			MinIdleConns: options.minIdleConns,
			TLSConfig:    options.tlsConfig,
		}), nil
	case ModeSentinel:
		if options.masterName == "" {
			return nil, fmt.Errorf("redisx: master name is required for sentinel mode")
		}
		if len(options.addrs) == 0 {
			return nil, fmt.Errorf("redisx: addrs are required for sentinel mode")
		}
		return redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    options.masterName,
			SentinelAddrs: options.addrs,
			Username:      options.username,
			Password:      options.password,
			DB:            options.db,
			DialTimeout:   options.dialTimeout,
			ReadTimeout:   options.readTimeout,
			WriteTimeout:  options.writeTimeout,
			PoolSize:      options.poolSize,
			MinIdleConns:  options.minIdleConns,
			TLSConfig:     options.tlsConfig,
		}), nil
	case ModeCluster:
		if len(options.addrs) == 0 {
			return nil, fmt.Errorf("redisx: addrs are required for cluster mode")
		}
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        options.addrs,
			Username:     options.username,
			Password:     options.password,
			DialTimeout:  options.dialTimeout,
			ReadTimeout:  options.readTimeout,
			WriteTimeout: options.writeTimeout,
			PoolSize:     options.poolSize,
			MinIdleConns: options.minIdleConns,
			TLSConfig:    options.tlsConfig,
		}), nil
	default:
		return nil, fmt.Errorf("redisx: unsupported mode %q", options.mode)
	}
}
