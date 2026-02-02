# config

基于 Viper 的通用配置加载器，特点是：

- 类型安全：`Load[T]` 直接反序列化到结构体
- 多来源：配置文件 + 环境变量（可选）+ 默认值（可选）
- 热更新：监听配置文件变更并触发回调（带 debounce）
- 并发安全：`Get()` 返回深拷贝，避免调用方误改内部状态

## 常见场景

### 1) 服务启动加载配置 + env 覆盖

```go
type AppConfig struct {
	Server struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"server"`
}

cfg, err := config.Load[AppConfig]("./config.yaml",
	config.WithDefaults[AppConfig](map[string]any{
		"server.host": "0.0.0.0",
		"server.port": 8080,
	}),
	config.WithEnv[AppConfig]("APP"), // APP_SERVER_HOST / APP_SERVER_PORT
)
if err != nil {
	log.Fatal(err)
}

current := cfg.Get()
_ = current
```

### 2) 热更新（重载 logger/server/client 等）

```go
cfg.OnChange(func(old, new AppConfig) {
	if config.Changed(old.Server, new.Server) {
		// restart server / update listener...
	}
})
```

### 3) JSON/YAML 示例

- YAML + defaults + env + watcher: `config/examples/yaml`
- JSON + watcher: `config/examples/json`

## API 速查

- `Load[T](path, ...Option[T]) (*Config[T], error)`
- `(*Config[T]).Get() T`
- `(*Config[T]).OnChange(func(old, new T))`
- `Changed(old, new T) bool`
- `WithDefaults(defaults map[string]any)`
- `WithEnv(prefix string)`

## 注意事项

- `Get()` 的深拷贝通过 JSON 序列化实现：适合配置这种“可 JSON 化”的结构体；不要在配置里放函数/通道/复杂自定义类型。

