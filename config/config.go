package config

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config 配置管理器
type Config[T any] struct {
	v        *viper.Viper
	value    *T
	mu       sync.RWMutex
	watchers []func(old, new T)
}

// Option 配置选项
type Option[T any] func(*Config[T])

// WithDefaults 设置默认值
func WithDefaults[T any](defaults map[string]any) Option[T] {
	return func(c *Config[T]) {
		for k, v := range defaults {
			c.v.SetDefault(k, v)
		}
	}
}

// WithEnv 绑定环境变量
func WithEnv[T any](prefix string) Option[T] {
	return func(c *Config[T]) {
		c.v.SetEnvPrefix(prefix)
		c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		c.v.AutomaticEnv()
	}
}

// Load 加载配置文件并自动监控变更
func Load[T any](path string, opts ...Option[T]) (*Config[T], error) {
	v := viper.New()
	v.SetConfigFile(path)

	c := &Config[T]{v: v}

	for _, opt := range opts {
		opt(c)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var val T
	if err := v.Unmarshal(&val); err != nil {
		return nil, err
	}
	c.value = &val

	c.watch()
	return c, nil
}

// Get 获取当前配置（并发安全，返回深拷贝）
func (c *Config[T]) Get() T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return deepCopy(*c.value)
}

// OnChange 注册配置变更回调
func (c *Config[T]) OnChange(callback func(old, new T)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.watchers = append(c.watchers, callback)
}

// Changed 比较两个值是否不同
func Changed[T any](old, new T) bool {
	return !reflect.DeepEqual(old, new)
}

// deepCopy 通过 JSON 序列化实现深拷贝
func deepCopy[T any](src T) T {
	var dst T
	data, _ := json.Marshal(src)
	_ = json.Unmarshal(data, &dst)
	return dst
}

func (c *Config[T]) watch() {
	var (
		debounceTimer *time.Timer
		debounceMu    sync.Mutex
	)

	c.v.OnConfigChange(func(_ fsnotify.Event) {
		debounceMu.Lock()
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
			c.handleConfigChange()
		})
		debounceMu.Unlock()
	})

	c.v.WatchConfig()
}

func (c *Config[T]) handleConfigChange() {
	oldConfig := c.Get()

	newConfig, watchers, ok := c.reloadConfig()
	if !ok {
		return
	}

	if reflect.DeepEqual(oldConfig, newConfig) {
		return
	}

	for _, cb := range watchers {
		func() {
			defer func() { _ = recover() }()
			cb(oldConfig, newConfig)
		}()
	}
}

// reloadConfig 重新加载配置，返回新配置、回调列表和是否成功
func (c *Config[T]) reloadConfig() (T, []func(old, new T), bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero T
	if err := c.v.ReadInConfig(); err != nil {
		return zero, nil, false
	}

	var val T
	if err := c.v.Unmarshal(&val); err != nil {
		return zero, nil, false
	}
	c.value = &val

	watchers := make([]func(old, new T), len(c.watchers))
	copy(watchers, c.watchers)

	return deepCopy(val), watchers, true
}
