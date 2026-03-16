# ESP8266 Universal Agent

这个目录提供一个可直接放到 GitHub Actions 构建的 ESP8266 通用固件工程，目标是让 `NodeMCU v2 / Wemos D1 mini` 一类板子能用同一套固件接入当前平台。

## 能力

- 单固件支持三种上行协议：`tcp / http / mqtt`
- 首次上电自动开启配网门户，保存 Wi-Fi 和平台参数
- 本地持久化配置到 `LittleFS`
- 本地状态页：`http://<device-ip>/`
- 本地 OTA 更新页：`http://<device-ip>/update`
- 传感器采集：
  - `BME280`
  - `BH1750`
  - `DS18B20`
  - `DHT11 / DHT22`
  - `A0` 模拟量
- TCP 模式支持平台命令回执
- MQTT 模式支持本地下行主题 `/down` 和 ACK 主题 `/ack`

## 支持板型

- `nodemcuv2`
- `d1_mini`

对应 PlatformIO 环境见 [platformio.ini](/f:/imx6ull/11建庄/codex/getdata/firmware/esp8266-universal/platformio.ini)。

## 默认接线

| 传感器 | 默认引脚 | 说明 |
| --- | --- | --- |
| I2C SDA | GPIO4 (`D2`) | BME280 / BH1750 |
| I2C SCL | GPIO5 (`D1`) | BME280 / BH1750 |
| DS18B20 | GPIO14 (`D5`) | 需要 4.7k 上拉 |
| DHT11/DHT22 | GPIO12 (`D6`) | 在配网门户里填 `dht11` 或 `dht22` |
| ADC | `A0` | 始终采集 |

## 一键接入流程

### 1. 先在平台创建产品和设备

推荐先选好产品接入模板，再创建一个设备，拿到：

- `device_id`
- `token`

如果你走：

- TCP：产品建议选 `tcp_json`
- HTTP：产品建议选 `http_json`
- MQTT：产品建议选 `mqtt_json`

详细见 [协议接入手册](/f:/imx6ull/11建庄/codex/getdata/doc/protocol-access-guide.md)。

### 2. 构建或下载二进制

本地构建：

```bash
cd firmware/esp8266-universal
pio run -e nodemcuv2
pio run -e d1_mini
```

构建产物：

- `.pio/build/nodemcuv2/firmware.bin`
- `.pio/build/d1_mini/firmware.bin`

仓库也提供 GitHub Actions，会直接产出这两个 `.bin`。

### 3. 烧录

PlatformIO：

```bash
cd firmware/esp8266-universal
pio run -e nodemcuv2 -t upload
```

或使用 `esptool.py` / NodeMCU Flasher 烧录生成的 `firmware.bin`。

### 4. 首次上电配网

首次启动会打开热点：

```text
MVP-ESP8266-<chipid>
```

密码：

```text
esp82666
```

手机或电脑连上这个热点后，打开：

```text
http://192.168.4.1
```

填入：

- Wi-Fi SSID / Password
- `Protocol`
- `Server Host`
- `TCP Port`
- `HTTP Port`
- `MQTT Port`
- `MQTT User / Password`（可选）
- `MQTT Topic`
- `HTTP Path`
- `Device ID`
- `Device Token`
- `Telemetry Interval ms`
- `DHT Pin`
- `DHT Type`
- `DS18B20 Pin`

保存后设备会自动联网并开始上报。

## 协议配置建议

### TCP 直连

适合：设备直接接平台，且需要命令下发。

推荐填写：

- `Protocol`: `tcp`
- `Server Host`: 平台地址
- `TCP Port`: `18830`

设备上报格式：

```json
{"type":"telemetry","values":{"temperature":24.8,"humidity":58}}
```

### HTTP Push

适合：设备周期上报，不要求平台主动下发命令。

推荐填写：

- `Protocol`: `http`
- `Server Host`: 平台地址
- `HTTP Port`: `8080`
- `HTTP Path`: `/api/v1/ingest/http/{device_id}`

设备上报格式：

```json
{
  "device_id":"dev_xxx",
  "token":"token_xxx",
  "protocol":"http_json",
  "values":{"temperature":24.8,"humidity":58}
}
```

### MQTT

适合：设备先进 Broker，再由桥接服务送平台。

推荐填写：

- `Protocol`: `mqtt`
- `Server Host`: MQTT Broker 地址
- `MQTT Port`: `1883`
- `MQTT Topic`: `mvp/{device_id}/up`

设备上报格式：

```json
{
  "device_id":"dev_xxx",
  "token":"token_xxx",
  "protocol":"mqtt_json",
  "values":{"temperature":24.8,"humidity":58}
}
```

Broker / Rule Engine / Node-RED 再把这段消息 POST 到平台：

```text
POST http://<platform-host>:8080/api/v1/ingest/http/<device_id>
```

## 固件自动采集字段

固件会根据实际检测到的传感器上报这些标准字段：

- `temperature`
- `humidity`
- `pressure`
- `illuminance`
- `probe_temperature`
- `dht_temperature`
- `dht_humidity`
- `analog`
- `rssi`
- `free_heap`
- `uptime_sec`
- `chip_id`
- `fw_version`

## 支持的命令

当前固件内置了这些命令：

- `reboot`
- `restart`
- `telemetry_once`
- `set_interval`
- `reset_config`

说明：

- 平台原生命令链路当前只在 `tcp` 模式下可直接打通
- `mqtt` 模式固件会监听 `<topic>/down` 并向 `<topic>/ack` 回 ACK，但平台侧还需要你自己做桥接

## 本地管理页

设备联网后，浏览器打开：

- `http://<device-ip>/`
- `http://<device-ip>/status`
- `http://<device-ip>/update`

其中：

- `/`：HTML 状态页
- `/status`：JSON 状态页
- `/update`：OTA 页面

OTA 页面默认账号：

- 用户名：`admin`
- 密码：设备 `token`

## GitHub Actions

仓库的固件工作流会构建：

- `esp8266-universal_nodemcuv2.bin`
- `esp8266-universal_d1_mini.bin`

你可以直接从 Actions Artifact 下载后烧录。

## 参考资料

- PlatformIO `nodemcuv2`：https://docs.platformio.org/en/latest/boards/espressif8266/nodemcuv2.html
- PlatformIO `d1_mini`：https://docs.platformio.org/en/latest/boards/espressif8266/d1_mini.html
- ESP8266 Arduino Core：https://arduino-esp8266.readthedocs.io/
- ESP8266 OTA：https://arduino-esp8266.readthedocs.io/en/latest/ota_updates/readme.html
- WiFiManager：https://github.com/tzapu/WiFiManager
- PubSubClient：https://github.com/knolleary/pubsubclient
- DHT Library：https://github.com/adafruit/DHT-sensor-library
- DallasTemperature：https://github.com/milesburton/Arduino-Temperature-Control-Library
- BH1750：https://github.com/claws/BH1750
- Adafruit BME280：https://github.com/adafruit/Adafruit_BME280_Library
