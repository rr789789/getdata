# 协议接入手册

这份手册说明当前 MVP 平台的实际接入边界、产品侧如何配置接入档案、不同协议的数据应如何整理后送入平台，以及 ESP8266 固件怎么选 `tcp / http / mqtt`。

## 1. 当前接入模型

当前平台分成两层：

- 原生入口层：平台当前真正内置的监听入口是 `TCP Gateway`、`HTTP Push` 和 `MQTT Broker`
- 协议适配层：`Modbus / OPC UA / BACnet / LoRaWAN` 通过“边缘采集器 / 协议桥 + HTTP Push”接入

可以直接这样理解：

| 来源 | 推荐链路 | 平台侧入口 | 当前状态 |
| --- | --- | --- | --- |
| ESP8266 / MCU 直连 | `tcp_json` | `:18830` | 已内置 |
| HTTP 能力设备 | `http_json` | `/api/v1/ingest/http/{device_id}` | 已内置 |
| MQTT 设备 | `mqtt_json` | `:1883` | 已内置 |
| Modbus RTU/TCP | `register_map -> Collector -> HTTP Push` | `/api/v1/ingest/http/{device_id}` | 通过桥接实现 |
| OPC UA | `node_map -> Collector -> HTTP Push` | `/api/v1/ingest/http/{device_id}` | 通过桥接实现 |
| BACnet | `object_map -> Collector -> HTTP Push` | `/api/v1/ingest/http/{device_id}` | 通过桥接实现 |
| LoRaWAN | `uplink decode -> HTTP Push` | `/api/v1/ingest/http/{device_id}` | 通过桥接实现 |

控制台里对应的产品接入模板在 [internal/model/access.go](/f:/imx6ull/11建庄/codex/getdata/internal/model/access.go)。

## 2. 接入前准备

无论你接什么协议，先做这三步：

1. 在控制台 `Product Center` 创建产品，或者通过 `POST /api/v1/products` 创建。
2. 给产品选一个合适的 `access_profile`：
   - `transport`
   - `protocol`
   - `ingest_mode`
   - `payload_format`
   - `point_mappings`
3. 注册设备，拿到：
   - `device_id`
   - `token`

查询协议目录：

```bash
curl http://127.0.0.1:8080/api/v1/protocol-catalog
```

## 3. TCP 直连接入

### 3.1 适用场景

- MCU/网关能直接维持 TCP 长连接
- 需要平台下发命令并等待设备回执
- 希望最少中间件

### 3.2 产品建议

```json
{
  "transport": "tcp",
  "protocol": "tcp_json",
  "ingest_mode": "gateway_tcp",
  "payload_format": "json_values",
  "auth_mode": "token"
}
```

### 3.3 协议格式

每条消息一行 JSON。

设备认证：

```json
{"type":"auth","device_id":"dev_xxx","token":"token_xxx"}
```

认证成功：

```json
{"type":"auth_ok","device_id":"dev_xxx","server_time":1710000000000}
```

设备心跳：

```json
{"type":"ping"}
```

设备遥测：

```json
{"type":"telemetry","values":{"temperature":25.3,"humidity":61}}
```

平台命令：

```json
{"type":"command","command_id":"cmd_xxx","name":"reboot","params":{"delay":1}}
```

设备回执：

```json
{"type":"ack","command_id":"cmd_xxx","status":"ok","message":"accepted"}
```

### 3.4 什么时候优先选 TCP

- 设备要接收平台命令
- 设备在线状态要强一致
- 同一设备上报频率高，不想每次都走 HTTP 建连

## 4. HTTP Push 接入

### 4.1 适用场景

- 设备只需要上报遥测
- 边缘采集器把多种协议统一转成 HTTP
- 云函数 / Webhook / LoRaWAN Network Server 回调

### 4.2 请求地址

```text
POST /api/v1/ingest/http/{device_id}
```

### 4.3 鉴权方式

当前支持三种 token 传法：

- Query：`?token=...`
- Header：`X-Device-Token: ...`
- Header：`Authorization: Bearer ...`
- Body：`{"token":"..."}`

### 4.4 直接 values 上报

```bash
curl -X POST http://127.0.0.1:8080/api/v1/ingest/http/<device_id> \
  -H "Content-Type: application/json" \
  -d '{
    "token":"<device_token>",
    "values":{"temperature":24.6,"humidity":56}
  }'
```

### 4.5 支持的载荷形状

当前服务端会依次解析：

- `values`
- `telemetry`
- `properties`
- `data`
- `registers`
- `nodes`
- `objects`
- 扁平 JSON

因此这些都能接：

```json
{"token":"xxx","values":{"temperature":25.1}}
```

```json
{"token":"xxx","telemetry":{"temperature":25.1}}
```

```json
{"token":"xxx","temperature":25.1,"humidity":60}
```

## 5. MQTT 接入

### 5.1 当前能力

当前平台已经内置 MQTT Broker，默认监听：

```text
:1883
```

设备可以直接连平台，不需要额外 Broker。

### 5.2 推荐做法

1. 设备使用 `device_id` 作为 MQTT `Username`
2. 设备使用 `device_token` 作为 MQTT `Password`
3. 设备向上行主题发布遥测
4. 平台向下行主题发布命令
5. 设备向 ACK 主题发布命令回执

### 5.3 推荐 MQTT Topic

默认主题：

```text
devices/{device_id}/up
devices/{device_id}/down
devices/{device_id}/ack
```

如果产品 `access_profile.topic` 指定了包含 `{device_id}` 的模板，例如：

```json
{
  "topic":"mvp/{device_id}/up"
}
```

平台会自动推导：

```text
mvp/{device_id}/up
mvp/{device_id}/down
mvp/{device_id}/ack
```

### 5.4 推荐 MQTT Payload

```json
{
  "values":{
    "temperature":24.5,
    "humidity":58
  }
}
```

ACK Payload：

```json
{
  "command_id":"cmd_xxx",
  "status":"ok",
  "message":"accepted"
}
```

### 5.5 产品建议

```json
{
  "transport":"mqtt",
  "protocol":"mqtt_json",
  "ingest_mode":"broker_mqtt",
  "payload_format":"json_values",
  "topic":"mvp/{device_id}/up"
}
```

### 5.6 命令链路说明

当前平台已经支持：

- 平台 -> MQTT 下行 Topic 的命令发布
- 设备 -> MQTT ACK Topic 的命令回执

因此 `MQTT` 现在已经是平台原生接入方式之一。

## 6. Modbus 接入

### 6.1 适用场景

- 温湿度变送器
- 电表/能耗表
- 压力/液位/流量仪表
- RS485 或 Modbus TCP 采集场景

### 6.2 推荐链路

```text
Modbus Sensor -> Edge Collector -> HTTP Push -> MVP Platform
```

### 6.3 产品配置

示例：

```json
{
  "transport":"rs485",
  "protocol":"modbus_rtu",
  "ingest_mode":"http_push",
  "payload_format":"register_map",
  "point_mappings":[
    {"source":"register:40001","property":"temperature","scale":0.1},
    {"source":"register:40002","property":"humidity","scale":0.1}
  ]
}
```

### 6.4 采集器上报格式

```json
{
  "token":"device_token",
  "registers":{
    "40001":231,
    "40002":556
  }
}
```

平台会根据 `point_mappings` 自动换算成：

```json
{
  "temperature": 23.1,
  "humidity": 55.6
}
```

## 7. OPC UA / BACnet / LoRaWAN

这三类协议的思路和 Modbus 一样，本质都是先在边缘做采集/解码，再通过 HTTP Push 送平台。

### 7.1 OPC UA

产品建议：

```json
{
  "transport":"opcua",
  "protocol":"opcua_json",
  "ingest_mode":"bridge_http",
  "payload_format":"mapped_json",
  "point_mappings":[
    {"source":"nodes.ns=2;s=Line1.Temperature","property":"temperature"},
    {"source":"nodes.ns=2;s=Line1.Running","property":"running"}
  ]
}
```

采集器格式：

```json
{
  "token":"device_token",
  "nodes":{
    "ns=2;s=Line1.Temperature":48.6,
    "ns=2;s=Line1.Running":true
  }
}
```

### 7.2 BACnet

产品建议：

```json
{
  "transport":"bacnet",
  "protocol":"bacnet_ip",
  "ingest_mode":"bridge_http",
  "payload_format":"mapped_json",
  "point_mappings":[
    {"source":"objects.analogInput:1.presentValue","property":"temperature"},
    {"source":"objects.analogInput:2.presentValue","property":"humidity"},
    {"source":"objects.analogInput:3.presentValue","property":"co2"}
  ]
}
```

采集器格式：

```json
{
  "token":"device_token",
  "objects":{
    "analogInput:1.presentValue":25.4,
    "analogInput:2.presentValue":55.8,
    "analogInput:3.presentValue":741
  }
}
```

### 7.3 LoRaWAN

LoRaWAN Network Server 一般会先把上行解码成 JSON，再通过 Webhook 转成：

```json
{
  "token":"device_token",
  "values":{
    "temperature":19.4,
    "humidity":67,
    "battery":88
  }
}
```

## 8. ESP8266 固件接入

仓库已经增加通用固件工程：

- [firmware/esp8266-universal/README.md](/f:/imx6ull/11建庄/codex/getdata/firmware/esp8266-universal/README.md)

它支持：

- 首次热点配网
- 持久化保存平台地址、`device_id`、`token`
- 协议选择：`tcp / http / mqtt`
- 本地 OTA 更新页
- 常见传感器采集：`BME280 / BH1750 / DS18B20 / DHT11/DHT22 / A0`

## 9. 排障建议

### 9.1 HTTP 返回 401

优先检查：

- `device_id` 是否正确
- `token` 是否与设备注册结果一致
- 请求体里是否有 `token`

### 9.2 设备在线但没遥测

- TCP 模式先确认是否收到 `auth_ok`
- HTTP 模式确认 `values` 或 `registers` 字段是否存在
- MQTT 模式确认桥接程序是否已经把消息 POST 到平台

### 9.3 MQTT 上报成功但平台没有命令下发

这是当前版本的预期边界。当前平台命令链路是 TCP 原生实现，MQTT 需要额外桥接。

### 9.4 Modbus 映射后字段为空

先检查：

- `source` 是否写成 `register:40001`
- `property` 是否存在于产品物模型
- `scale` 是否正确

## 10. 安全建议

- 设备 `token` 视为密钥，不要写死到公开仓库
- 边缘采集器和平台之间优先放在内网
- 生产环境建议在反向代理层补上 HTTPS 和鉴权审计
- MQTT Broker 建议单独做 ACL，不要默认匿名开放

## 参考资料

- ESP8266 Arduino Core 文档：https://arduino-esp8266.readthedocs.io/
- ESP8266 Filesystem / LittleFS：https://arduino-esp8266.readthedocs.io/en/latest/filesystem.html
- ESP8266 OTA 文档：https://arduino-esp8266.readthedocs.io/en/latest/ota_updates/readme.html
- WiFiManager 项目：https://github.com/tzapu/WiFiManager
- PubSubClient 项目：https://github.com/knolleary/pubsubclient
