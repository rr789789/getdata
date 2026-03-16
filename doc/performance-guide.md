# 性能与基准测试

这份说明覆盖三类验证：

- 基准测试：看核心方法和 HTTP 接入路径的单机热点
- 压测：通过内置 `mvp-loadgen` 对 `HTTP / TCP / MQTT` 做并发连接和消息发送
- 指标观测：通过 `/metrics` 或 `/metrics?format=prometheus` 看吞吐、连接、持久化和运行时状态

## 1. 运行前建议

推荐先区分两种场景：

- 吞吐场景：`MVP_STORE_BACKEND=memory`
  目的：排除本地 JSON 落盘带来的 I/O 开销，先看协议接入和内存路径吞吐
- 可靠性场景：`MVP_STORE_BACKEND=file`
  目的：验证本地快照持久化、重启恢复和指标里的落盘状态

示例：

```powershell
$env:MVP_STORE_BACKEND="memory"
$env:MVP_HTTP_ADDR=":8080"
$env:MVP_GATEWAY_ADDR=":18830"
$env:MVP_MQTT_ADDR=":1883"
go run .\cmd\mvp-platform
```

## 2. 基准测试

仓库内已经补充：

- `internal/core/service_benchmark_test.go`
- `internal/api/ingest_benchmark_test.go`

运行：

```powershell
go test -run=^$ -bench . -benchmem ./internal/core ./internal/api
```

如果你只看某一项：

```powershell
go test -run=^$ -bench BenchmarkHandleTelemetry -benchmem ./internal/core
go test -run=^$ -bench BenchmarkHTTPIngest -benchmem ./internal/api
```

建议重点关注：

- `ns/op`：单次调用耗时
- `B/op`：每次操作分配的字节数
- `allocs/op`：每次操作的内存分配次数

## 3. 压测工具

仓库新增了内置压测工具：

```text
cmd/mvp-loadgen
```

构建：

```powershell
go build -o .\bin\mvp-loadgen.exe .\cmd\mvp-loadgen
```

它会：

- 自动创建一个测试产品，除非显式传入 `-product-id`
- 自动注册一批测试设备
- 按指定模式对 `HTTP / TCP / MQTT` 发送遥测
- 最终输出一份 JSON 摘要

### 3.1 HTTP 压测

```powershell
.\bin\mvp-loadgen.exe `
  -mode http `
  -base-url http://127.0.0.1:8080 `
  -devices 500 `
  -messages 200 `
  -concurrency 64
```

### 3.2 TCP 压测

```powershell
.\bin\mvp-loadgen.exe `
  -mode tcp `
  -base-url http://127.0.0.1:8080 `
  -gateway-addr 127.0.0.1:18830 `
  -devices 1000 `
  -messages 100 `
  -concurrency 128
```

### 3.3 MQTT 压测

```powershell
.\bin\mvp-loadgen.exe `
  -mode mqtt `
  -base-url http://127.0.0.1:8080 `
  -mqtt-addr 127.0.0.1:1883 `
  -topic-template devices/{device_id}/up `
  -devices 1000 `
  -messages 100 `
  -concurrency 128
```

如果你的产品用了自定义 Topic 模板：

```powershell
.\bin\mvp-loadgen.exe `
  -mode mqtt `
  -product-id <existing_product_id> `
  -topic-template mvp/{device_id}/up `
  -devices 200 `
  -messages 300
```

### 3.4 参数说明

- `-mode`：`http | tcp | mqtt`
- `-devices`：注册的设备数
- `-messages`：每个设备发送多少条遥测
- `-concurrency`：同时活跃的设备数
- `-interval`：每条遥测之间的间隔，例如 `100ms`
- `-product-id`：复用已有产品
- `-topic-template`：MQTT 上行 Topic 模板，默认 `devices/{device_id}/up`

## 4. 观测指标

JSON 指标：

```text
http://127.0.0.1:8080/metrics
```

Prometheus 文本：

```text
http://127.0.0.1:8080/metrics?format=prometheus
```

建议重点看这些指标：

- `mvp_http_ingest_accepted_total`
- `mvp_tcp_telemetry_accepted_total`
- `mvp_mqtt_messages_received_total`
- `mvp_bytes_ingested_total`
- `mvp_telemetry_values_total`
- `mvp_transport_tcp_online_devices`
- `mvp_transport_mqtt_online_devices`
- `mvp_storage_persist_errors_total`
- `mvp_storage_last_persisted_unix`
- `mvp_runtime_goroutines`
- `mvp_runtime_heap_alloc_bytes`

## 5. 当前结论边界

这套压测工具和基准测试的作用是帮助你在当前 MVP 上快速获得：

- 单进程吞吐上限趋势
- 协议入口之间的相对差异
- 开启持久化前后的性能差异
- 运行时内存和协程数量变化

但它不代表已经完成了：

- 200k 长连接实测认证
- 多节点集群横向扩容
- 真正的分布式持久化和消息总线

如果目标是稳定走到 200k 设备，需要在下一阶段继续补：

- 外部数据库或时序库
- MQTT 集群或消息总线
- 接入层与控制面拆分
- 分片路由与批量写入
