# MVP IoT Platform

一个面向设备接入场景的最小可运行平台，目标是先打通“设备注册 -> 长连接接入 -> 遥测上报 -> 命令下发 -> 命令回执”的基础链路，并保留后续扩展到 200k 设备的结构空间。

## 当前能力

- HTTP 管理 API
- TCP 长连接设备网关
- 设备注册与 token 鉴权
- 遥测上报
- 命令下发
- 命令回执
- 健康检查与基础指标
- GitHub Actions 交叉编译 Windows / Linux 二进制

## 当前限制

- 目前是单节点单二进制
- 默认使用内存存储，进程重启后状态不保留
- 设备协议是 `TCP + JSON Lines`，还没有接入 MQTT
- 200k 设备接入是架构目标，不是当前单节点版本的实测结论

## 文档

- 方案计划：[doc/mvp-platform-plan.md](doc/mvp-platform-plan.md)

## 本地运行

```bash
go test ./...
go build -o bin/mvp-platform ./cmd/mvp-platform
./bin/mvp-platform
```

Windows:

```powershell
go test ./...
go build -o bin\mvp-platform.exe .\cmd\mvp-platform
.\bin\mvp-platform.exe
```

默认端口：

- HTTP API: `:8080`
- Device Gateway: `:18830`

## 配置项

通过环境变量配置：

- `MVP_HTTP_ADDR`，默认 `:8080`
- `MVP_GATEWAY_ADDR`，默认 `:18830`
- `MVP_LOG_LEVEL`，默认 `info`
- `MVP_SHUTDOWN_TIMEOUT`，默认 `10s`
- `MVP_DEVICE_AUTH_TIMEOUT`，默认 `15s`
- `MVP_DEVICE_IDLE_TIMEOUT`，默认 `90s`
- `MVP_DEVICE_WRITE_TIMEOUT`，默认 `5s`
- `MVP_DEVICE_QUEUE_SIZE`，默认 `128`
- `MVP_TELEMETRY_RETENTION`，默认 `200`
- `MVP_MAX_MESSAGE_BYTES`，默认 `1048576`

## HTTP API

注册设备：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{"name":"demo-device","metadata":{"site":"lab"}}'
```

查询设备：

```bash
curl http://127.0.0.1:8080/api/v1/devices/<device_id>
```

查询遥测：

```bash
curl http://127.0.0.1:8080/api/v1/devices/<device_id>/telemetry?limit=20
```

下发命令：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/devices/<device_id>/commands \
  -H "Content-Type: application/json" \
  -d '{"name":"reboot","params":{"delay":1}}'
```

健康检查：

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/metrics
```

## 设备协议

协议格式：每条消息一行 JSON。

设备认证：

```json
{"type":"auth","device_id":"dev_xxx","token":"token_xxx"}
```

平台认证成功响应：

```json
{"type":"auth_ok","device_id":"dev_xxx","server_time":1710000000000}
```

设备心跳：

```json
{"type":"ping"}
```

平台心跳响应：

```json
{"type":"pong","server_time":1710000000000}
```

设备遥测：

```json
{"type":"telemetry","ts":1710000000000,"values":{"temp":25.3,"hum":61}}
```

平台下发命令：

```json
{"type":"command","command_id":"cmd_xxx","name":"reboot","params":{"delay":1}}
```

设备命令回执：

```json
{"type":"ack","command_id":"cmd_xxx","status":"ok","message":"accepted"}
```

## GitHub Actions

仓库内已提供 `.github/workflows/build.yml`：

- 自动执行 `go test ./...`
- 交叉编译 `linux/windows + amd64/arm64`
- 上传构建产物为 Actions artifact

## 下一步建议

- 接入持久化存储
- 增加压测和 benchmark
- 增加 MQTT 适配层
- 拆分 gateway / core / storage
