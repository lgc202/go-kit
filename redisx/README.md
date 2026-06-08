# redisx

`redisx` 是基于 `github.com/redis/go-redis/v9` 的 Redis client 初始化包。
它只处理连接参数和常见部署模式，不封装 Redis 命令。

## 支持的部署模式

- 单机 Redis
- Redis Sentinel
- Redis Cluster

## 单机

```go
client, err := redisx.New(redisx.WithAddr("127.0.0.1:6379"))
if err != nil {
	return err
}
defer client.Close()
```

## Sentinel

```go
client, err := redisx.New(
	redisx.WithSentinel("mymaster", "10.0.0.1:26379", "10.0.0.2:26379"),
)
if err != nil {
	return err
}
defer client.Close()
```

## Cluster

```go
client, err := redisx.New(
	redisx.WithCluster("10.0.0.1:6379", "10.0.0.2:6379"),
)
if err != nil {
	return err
}
defer client.Close()
```

## 连通性检查

默认只创建 client，不访问 Redis。需要启动时检查连通性时，设置 `Ping`：

```go
client, err := redisx.New(
	redisx.WithAddr("127.0.0.1:6379"),
	redisx.WithPing(true),
)
```
