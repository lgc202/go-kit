# redisx

基于 `go-redis/v9` 的 Redis client 初始化包，支持单机、Sentinel、Cluster。

```go
client, err := redisx.New(redisx.WithAddr("127.0.0.1:6379"))
if err != nil {
	return err
}
defer client.Close()
```

Sentinel / Cluster：

```go
client, err := redisx.New(redisx.WithSentinel("mymaster", "10.0.0.1:26379"))
client, err = redisx.New(redisx.WithCluster("10.0.0.1:6379", "10.0.0.2:6379"))
```
