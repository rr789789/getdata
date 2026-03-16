# MVP IoT Platform

一个面向设备接入场景的最小可运行平台，目标是先打通“设备注册 -> 长连接接入 -> 遥测上报 -> 命令下发 -> 命令回执”的基础链路，并保留后续扩展到 200k 设备的结构空间。

## 当前能力

- 内嵌网页后台
- HTTP 管理 API
- TCP 长连接设备网关
- 设备注册与 token 鉴权
- 遥测上报
- 命令下发
- 命令回执
- 带 UI 的测试设备模拟器
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

启动后直接打开：

- 管理台: `http://127.0.0.1:8080/`
- 健康检查: `http://127.0.0.1:8080/healthz`
- 运行指标: `http://127.0.0.1:8080/metrics`

## 配置项

通过环境变量配置：

- `MVP_HTTP_ADDR`，默认 `:8080`
- `MVP_GATEWAY_ADDR`，默认 `:18830`
- `MVP_GATEWAY_DIAL_ADDR`，默认从 `MVP_GATEWAY_ADDR` 推导，通常是 `127.0.0.1:18830`
- `MVP_LOG_LEVEL`，默认 `info`
- `MVP_SHUTDOWN_TIMEOUT`，默认 `10s`
- `MVP_DEVICE_AUTH_TIMEOUT`，默认 `15s`
- `MVP_DEVICE_IDLE_TIMEOUT`，默认 `90s`
- `MVP_DEVICE_WRITE_TIMEOUT`，默认 `5s`
- `MVP_DEVICE_QUEUE_SIZE`，默认 `128`
- `MVP_TELEMETRY_RETENTION`，默认 `200`
- `MVP_MAX_MESSAGE_BYTES`，默认 `1048576`

说明：

- `MVP_GATEWAY_ADDR` 是设备网关监听地址
- `MVP_GATEWAY_DIAL_ADDR` 是内置模拟器拨号到设备网关时使用的地址
- 如果你把网关监听到 `0.0.0.0:18830`，内置模拟器通常仍然使用 `127.0.0.1:18830`

## 网页后台

打开 `http://127.0.0.1:8080/` 后，可以直接在页面完成：

- 注册设备
- 查看设备在线状态
- 查看最近遥测
- 下发命令
- 创建测试模拟器
- 控制模拟器连接 / 断开 / 手动发遥测
- 查看模拟器日志

这个管理台是内嵌在同一个 `exe` 里的，不需要单独前端构建。

## 内置模拟器

内置模拟器不是伪造 API 数据，而是由后端真实建立 TCP 连接到设备网关，按平台协议执行：

- `auth`
- `ping`
- `telemetry`
- 自动接收 `command`
- 自动发送 `ack`

因此它适合用来验证完整的设备接入链路。

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

模拟器 API：

```bash
curl http://127.0.0.1:8080/api/v1/simulators
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
