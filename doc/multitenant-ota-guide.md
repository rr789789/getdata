# 多租户 / 规则引擎 / OTA 使用说明

## 概览

当前版本新增了四块控制面能力：

- 多租户：`/api/v1/tenants`
- 规则动作：`alert / send_command / apply_config_profile`
- 固件仓库：`/api/v1/firmware`
- OTA 任务：`/api/v1/ota-campaigns`

网页控制台入口仍然是：

- `http://127.0.0.1:8080/`

控制台里新增了：

- 顶部 `Tenant Filter`
- `Product Center` 里的 `Tenant Workspace`
- `Governance` 里的规则动作表单
- `Config Center` 里的 `Firmware Repository` 和 `OTA Campaigns`

## 推荐操作顺序

1. 创建租户
2. 选择顶部租户筛选
3. 在该租户下创建产品
4. 创建设备 / 分组 / 配置模板
5. 创建规则动作
6. 上传固件元数据
7. 创建 OTA 任务

## 租户 API

创建租户：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Factory East",
    "slug":"factory-east",
    "description":"Factory East business unit",
    "metadata":{"region":"cn-east","tier":"production"}
  }'
```

列出租户：

```bash
curl http://127.0.0.1:8080/api/v1/tenants
```

## 租户资源隔离

以下资源都支持 `tenant_id`：

- `/api/v1/products`
- `/api/v1/devices`
- `/api/v1/groups`
- `/api/v1/rules`
- `/api/v1/alerts`
- `/api/v1/config-profiles`
- `/api/v1/firmware`
- `/api/v1/ota-campaigns`

示例：

```bash
curl "http://127.0.0.1:8080/api/v1/products?tenant_id=<tenant_id>"
curl "http://127.0.0.1:8080/api/v1/alerts?tenant_id=<tenant_id>&limit=20"
```

## 规则动作

### 1. 告警动作

```json
{
  "type":"alert",
  "severity":"warning",
  "message":"temperature is over threshold"
}
```

### 2. 自动命令动作

```json
{
  "type":"send_command",
  "name":"reboot",
  "params":{"delay":1}
}
```

### 3. 应用配置模板动作

```json
{
  "type":"apply_config_profile",
  "config_profile_id":"<profile_id>"
}
```

创建带动作的规则：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id":"<tenant_id>",
    "name":"temp-high",
    "product_id":"<product_id>",
    "device_id":"<device_id>",
    "severity":"warning",
    "cooldown_seconds":60,
    "condition":{"property":"temperature","operator":"gt","value":30},
    "actions":[
      {"type":"send_command","name":"reboot","params":{"delay":1}},
      {"type":"apply_config_profile","config_profile_id":"<profile_id>"}
    ]
  }'
```

## 固件仓库

创建固件制品：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/firmware \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id":"<tenant_id>",
    "product_id":"<product_id>",
    "name":"esp8266-universal",
    "version":"1.0.0",
    "file_name":"esp8266-universal.bin",
    "url":"https://example.com/firmware/esp8266-universal.bin",
    "checksum":"<sha256>",
    "checksum_type":"sha256",
    "size_bytes":524288
  }'
```

查询固件：

```bash
curl "http://127.0.0.1:8080/api/v1/firmware?tenant_id=<tenant_id>"
```

## OTA 任务

创建 OTA 任务时可选三种范围：

- `product_id`
- `group_id`
- `device_id`

最小单设备示例：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/ota-campaigns \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id":"<tenant_id>",
    "name":"east-rollout",
    "firmware_id":"<firmware_id>",
    "device_id":"<device_id>"
  }'
```

查询 OTA：

```bash
curl "http://127.0.0.1:8080/api/v1/ota-campaigns?tenant_id=<tenant_id>"
```

## OTA 下发载荷

当前 OTA 任务会向设备下发一条 `ota_upgrade` 命令，核心字段包含：

- `campaign_id`
- `firmware_id`
- `name`
- `version`
- `url`
- `checksum`
- `checksum_type`
- `file_name`
- `size_bytes`

设备执行完成后，需要按已有命令回执链路回 `ack`，平台会据此更新：

- `acked_count`
- `failed_count`
- `status`

## 当前边界

- 当前是单节点 MVP，不是完整集群版
- OTA 这里存的是固件元数据和任务，不负责对象存储分发
- 规则引擎当前是阈值规则，不是可视化流程编排器
- 多租户当前是资源隔离模型，不包含用户 / RBAC / 审计
