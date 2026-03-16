# MVP IoT Platform

一个面向设备接入场景的最小可运行物联网平台。

当前版本已经覆盖一条完整的基础链路：

- 产品定义
- 物模型（Thing Model / TSL）
- 设备注册与 `token` 鉴权
- TCP 长连接接入
- 遥测上报
- 设备影子
- 命令下发与回执
- 静态设备分组
- 阈值规则与告警事件
- 带 UI 的测试设备模拟器
- 内嵌网页控制台
- GitHub Actions 交叉编译 Windows / Linux 二进制

## 文档

- 方案计划：[doc/mvp-platform-plan.md](doc/mvp-platform-plan.md)

## 当前限制

- 当前是单节点单二进制 MVP
- 默认使用内存存储，进程重启后状态不会保留
- 设备协议当前是 `TCP + JSON Lines`
- 200k 设备接入目前是架构目标，不是这版单节点的实测结论

## 本地运行

Linux / macOS:

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

- 控制台：`http://127.0.0.1:8080/`
- 健康检查：`http://127.0.0.1:8080/healthz`
- 指标：`http://127.0.0.1:8080/metrics`

## 环境变量

- `MVP_HTTP_ADDR`，默认 `:8080`
- `MVP_GATEWAY_ADDR`，默认 `:18830`
- `MVP_GATEWAY_DIAL_ADDR`，默认从 `MVP_GATEWAY_ADDR` 推导，通常为 `127.0.0.1:18830`
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
- 如果网关监听 `0.0.0.0:18830`，内置模拟器通常仍使用 `127.0.0.1:18830`

## 控制台能力

打开 `http://127.0.0.1:8080/` 后，可以直接完成：

- 创建产品和物模型
- 注册设备并绑定产品
- 查看设备在线状态、遥测、命令、影子
- 创建静态设备分组并把设备加入 / 移出分组
- 创建按产品 / 分组 / 设备范围生效的阈值规则
- 查看最近告警事件
- 创建和控制测试设备模拟器

## 典型 API

创建产品：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "name":"thermostat-product",
    "description":"demo product",
    "thing_model":{
      "properties":[
        {"identifier":"temperature","name":"Temperature","data_type":"float","access_mode":"rw"}
      ],
      "services":[
        {"identifier":"reboot","name":"Reboot"}
      ]
    }
  }'
```

注册设备：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{"name":"device-01","product_id":"<product_id>","metadata":{"site":"lab"}}'
```

创建分组：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/groups \
  -H "Content-Type: application/json" \
  -d '{"name":"line-a","product_id":"<product_id>","tags":{"site":"factory-1"}}'
```

把设备加入分组：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/groups/<group_id>/devices \
  -H "Content-Type: application/json" \
  -d '{"device_id":"<device_id>"}'
```

创建规则：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name":"temp-high",
    "product_id":"<product_id>",
    "group_id":"<group_id>",
    "severity":"critical",
    "cooldown_seconds":60,
    "condition":{
      "property":"temperature",
      "operator":"gt",
      "value":30
    }
  }'
```

查询最近告警：

```bash
curl "http://127.0.0.1:8080/api/v1/alerts?limit=20"
```

查询设备影子：

```bash
curl http://127.0.0.1:8080/api/v1/devices/<device_id>/shadow
```

更新设备期望影子：

```bash
curl -X PUT http://127.0.0.1:8080/api/v1/devices/<device_id>/shadow \
  -H "Content-Type: application/json" \
  -d '{"desired":{"temperature":26.5}}'
```

下发命令：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/devices/<device_id>/commands \
  -H "Content-Type: application/json" \
  -d '{"name":"reboot","params":{"delay":1}}'
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
{"type":"telemetry","ts":1710000000000,"values":{"temperature":25.3,"humidity":61}}
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

仓库已提供 `.github/workflows/build.yml`：

- 自动执行 `go test ./...`
- 交叉编译 `linux/windows + amd64/arm64`
- 上传构建产物到 Actions artifact

## 后续建议

- 接入持久化存储
- 增加 MQTT 适配层
- 规则动作扩展到自动命令 / Webhook
- 增加分页、实时推送和审计日志
- 拆分 gateway / core / storage
