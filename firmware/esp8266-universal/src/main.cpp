#include <Arduino.h>
#include <ArduinoJson.h>
#include <BH1750.h>
#include <DHT.h>
#include <DallasTemperature.h>
#include <ESP8266HTTPClient.h>
#include <ESP8266HTTPUpdateServer.h>
#include <ESP8266WebServer.h>
#include <ESP8266WiFi.h>
#include <ESP8266mDNS.h>
#include <FS.h>
#include <LittleFS.h>
#include <OneWire.h>
#include <PubSubClient.h>
#include <Wire.h>
#include <WiFiManager.h>
#include <Adafruit_BME280.h>

#ifndef FW_VERSION
#define FW_VERSION "0.1.0"
#endif

#ifndef DEFAULT_TELEMETRY_INTERVAL_MS
#define DEFAULT_TELEMETRY_INTERVAL_MS 15000UL
#endif

#ifndef DEFAULT_TCP_PORT
#define DEFAULT_TCP_PORT 18830
#endif

#ifndef DEFAULT_HTTP_PORT
#define DEFAULT_HTTP_PORT 8080
#endif

#ifndef DEFAULT_MQTT_PORT
#define DEFAULT_MQTT_PORT 1883
#endif

#ifndef DEFAULT_I2C_SDA
#define DEFAULT_I2C_SDA 4
#endif

#ifndef DEFAULT_I2C_SCL
#define DEFAULT_I2C_SCL 5
#endif

#ifndef DEFAULT_DHT_PIN
#define DEFAULT_DHT_PIN 12
#endif

#ifndef DEFAULT_ONEWIRE_PIN
#define DEFAULT_ONEWIRE_PIN 14
#endif

#ifndef DEFAULT_HTTP_PATH
#define DEFAULT_HTTP_PATH "/api/v1/ingest/http/{device_id}"
#endif

namespace {

const char *CONFIG_PATH = "/config.json";
const char *PORTAL_PASSWORD = "esp82666";
const unsigned long RETRY_BACKOFF_MS = 5000UL;
const unsigned long TCP_PING_INTERVAL_MS = 30000UL;
const unsigned long RESTART_DELAY_MS = 1200UL;
const size_t SMALL_JSON_BYTES = 1024;
const size_t MEDIUM_JSON_BYTES = 2048;

enum class TransportMode {
  TCP,
  HTTP,
  MQTT,
};

struct DeviceConfig {
  String protocol;
  String serverHost;
  uint16_t tcpPort;
  uint16_t httpPort;
  uint16_t mqttPort;
  String mqttUser;
  String mqttPassword;
  String mqttTopic;
  String httpPath;
  String deviceId;
  String deviceToken;
  unsigned long telemetryIntervalMs;
  bool enableAnalog;
  uint8_t dhtPin;
  String dhtType;
  uint8_t oneWirePin;
  uint8_t i2cSda;
  uint8_t i2cScl;
};

struct SensorState {
  bool bme280Active = false;
  bool bh1750Active = false;
  bool ds18b20Active = false;
  bool dhtActive = false;
  uint8_t ds18b20Count = 0;
};

struct RuntimeState {
  bool tcpAuthenticated = false;
  bool mdnsReady = false;
  bool adminServerStarted = false;
  bool forceTelemetry = false;
  bool restartRequested = false;
  unsigned long restartAtMs = 0;
  unsigned long lastTelemetryAtMs = 0;
  unsigned long lastTcpPingAtMs = 0;
  unsigned long lastConnectAttemptAtMs = 0;
  String lastError;
  String lastCommand;
  String lastAckStatus;
};

DeviceConfig gConfig;
SensorState gSensors;
RuntimeState gRuntime;

ESP8266WebServer gAdminServer(80);
ESP8266HTTPUpdateServer gHttpUpdater;
WiFiClient gTcpClient;
WiFiClient gHttpClient;
WiFiClient gMqttNetworkClient;
PubSubClient gMqttClient(gMqttNetworkClient);
Adafruit_BME280 gBme280;
BH1750 gBh1750;
OneWire *gOneWire = nullptr;
DallasTemperature *gDallas = nullptr;
DHT *gDht = nullptr;

String gHostname;
String gUpdaterPassword;
String gTcpReadBuffer;
bool gShouldSaveConfig = false;

String lowerCopy(const String &value) {
  String copy = value;
  copy.trim();
  copy.toLowerCase();
  return copy;
}

String chipIdHex() {
  char buffer[9];
  snprintf(buffer, sizeof(buffer), "%06X", ESP.getChipId());
  return String(buffer);
}

String fallbackDeviceId() {
  return "esp8266-" + chipIdHex();
}

String defaultHostname() {
  return "mvp-esp8266-" + chipIdHex();
}

String defaultMqttTopic() {
  return "mvp/{device_id}/up";
}

uint16_t normalizedPort(long value, uint16_t fallback) {
  if (value <= 0 || value > 65535) {
    return fallback;
  }
  return static_cast<uint16_t>(value);
}

uint8_t normalizedPin(long value, uint8_t fallback) {
  if (value < 0 || value > 16) {
    return fallback;
  }
  return static_cast<uint8_t>(value);
}

unsigned long normalizedInterval(unsigned long value) {
  if (value < 5000UL) {
    return 5000UL;
  }
  if (value > 3600000UL) {
    return 3600000UL;
  }
  return value;
}

bool parseBoolFlag(const String &value, bool fallback) {
  const String normalized = lowerCopy(value);
  if (normalized == "1" || normalized == "true" || normalized == "yes" || normalized == "on") {
    return true;
  }
  if (normalized == "0" || normalized == "false" || normalized == "no" || normalized == "off") {
    return false;
  }
  return fallback;
}

String maskedSecret(const String &value) {
  if (value.length() <= 4) {
    return "****";
  }
  return value.substring(0, 2) + "..." + value.substring(value.length() - 2);
}

void logLine(const String &message) {
  Serial.println("[mvp-esp8266] " + message);
}

String resolvePlaceholders(String value) {
  value.replace("{device_id}", gConfig.deviceId);
  value.replace("{chip_id}", chipIdHex());
  return value;
}

TransportMode currentTransport() {
  const String protocol = lowerCopy(gConfig.protocol);
  if (protocol == "http") {
    return TransportMode::HTTP;
  }
  if (protocol == "mqtt") {
    return TransportMode::MQTT;
  }
  return TransportMode::TCP;
}

String mqttUpTopic() {
  String topic = resolvePlaceholders(gConfig.mqttTopic.length() ? gConfig.mqttTopic : defaultMqttTopic());
  if (!topic.endsWith("/up")) {
    topic += "/up";
  }
  return topic;
}

String mqttBaseTopic() {
  String topic = mqttUpTopic();
  if (topic.endsWith("/up")) {
    topic.remove(topic.length() - 3);
  }
  return topic;
}

String mqttDownTopic() {
  return mqttBaseTopic() + "/down";
}

String mqttAckTopic() {
  return mqttBaseTopic() + "/ack";
}

String httpIngestPath() {
  return resolvePlaceholders(gConfig.httpPath.length() ? gConfig.httpPath : String(DEFAULT_HTTP_PATH));
}

void setDefaultConfig() {
  gConfig.protocol = "tcp";
  gConfig.serverHost = "192.168.1.10";
  gConfig.tcpPort = DEFAULT_TCP_PORT;
  gConfig.httpPort = DEFAULT_HTTP_PORT;
  gConfig.mqttPort = DEFAULT_MQTT_PORT;
  gConfig.mqttUser = "";
  gConfig.mqttPassword = "";
  gConfig.mqttTopic = defaultMqttTopic();
  gConfig.httpPath = DEFAULT_HTTP_PATH;
  gConfig.deviceId = fallbackDeviceId();
  gConfig.deviceToken = "";
  gConfig.telemetryIntervalMs = DEFAULT_TELEMETRY_INTERVAL_MS;
  gConfig.enableAnalog = true;
  gConfig.dhtPin = DEFAULT_DHT_PIN;
  gConfig.dhtType = "none";
  gConfig.oneWirePin = DEFAULT_ONEWIRE_PIN;
  gConfig.i2cSda = DEFAULT_I2C_SDA;
  gConfig.i2cScl = DEFAULT_I2C_SCL;
}

bool isConfigValid() {
  return gConfig.serverHost.length() > 0 && gConfig.deviceId.length() > 0 && gConfig.deviceToken.length() > 0;
}

bool loadConfig() {
  setDefaultConfig();
  if (!LittleFS.exists(CONFIG_PATH)) {
    return false;
  }

  File file = LittleFS.open(CONFIG_PATH, "r");
  if (!file) {
    gRuntime.lastError = "failed to open config file";
    return false;
  }

  DynamicJsonDocument doc(MEDIUM_JSON_BYTES);
  DeserializationError error = deserializeJson(doc, file);
  file.close();
  if (error) {
    gRuntime.lastError = String("failed to parse config: ") + error.c_str();
    return false;
  }

  gConfig.protocol = doc["protocol"] | gConfig.protocol;
  gConfig.serverHost = doc["server_host"] | gConfig.serverHost;
  gConfig.tcpPort = normalizedPort(doc["tcp_port"] | gConfig.tcpPort, DEFAULT_TCP_PORT);
  gConfig.httpPort = normalizedPort(doc["http_port"] | gConfig.httpPort, DEFAULT_HTTP_PORT);
  gConfig.mqttPort = normalizedPort(doc["mqtt_port"] | gConfig.mqttPort, DEFAULT_MQTT_PORT);
  gConfig.mqttUser = doc["mqtt_user"] | gConfig.mqttUser;
  gConfig.mqttPassword = doc["mqtt_password"] | gConfig.mqttPassword;
  gConfig.mqttTopic = doc["mqtt_topic"] | gConfig.mqttTopic;
  gConfig.httpPath = doc["http_path"] | gConfig.httpPath;
  gConfig.deviceId = doc["device_id"] | gConfig.deviceId;
  gConfig.deviceToken = doc["device_token"] | gConfig.deviceToken;
  gConfig.telemetryIntervalMs = normalizedInterval(doc["telemetry_interval_ms"] | gConfig.telemetryIntervalMs);
  gConfig.enableAnalog = doc["enable_analog"] | gConfig.enableAnalog;
  gConfig.dhtPin = normalizedPin(doc["dht_pin"] | gConfig.dhtPin, DEFAULT_DHT_PIN);
  gConfig.dhtType = doc["dht_type"] | gConfig.dhtType;
  gConfig.oneWirePin = normalizedPin(doc["onewire_pin"] | gConfig.oneWirePin, DEFAULT_ONEWIRE_PIN);
  gConfig.i2cSda = normalizedPin(doc["i2c_sda"] | gConfig.i2cSda, DEFAULT_I2C_SDA);
  gConfig.i2cScl = normalizedPin(doc["i2c_scl"] | gConfig.i2cScl, DEFAULT_I2C_SCL);

  gConfig.protocol.trim();
  gConfig.serverHost.trim();
  gConfig.mqttTopic.trim();
  gConfig.httpPath.trim();
  gConfig.deviceId.trim();
  gConfig.deviceToken.trim();
  gConfig.dhtType.trim();
  if (gConfig.deviceId.isEmpty()) {
    gConfig.deviceId = fallbackDeviceId();
  }
  if (gConfig.mqttTopic.isEmpty()) {
    gConfig.mqttTopic = defaultMqttTopic();
  }
  if (gConfig.httpPath.isEmpty()) {
    gConfig.httpPath = DEFAULT_HTTP_PATH;
  }
  return true;
}

bool saveConfig() {
  DynamicJsonDocument doc(MEDIUM_JSON_BYTES);
  doc["protocol"] = lowerCopy(gConfig.protocol);
  doc["server_host"] = gConfig.serverHost;
  doc["tcp_port"] = gConfig.tcpPort;
  doc["http_port"] = gConfig.httpPort;
  doc["mqtt_port"] = gConfig.mqttPort;
  doc["mqtt_user"] = gConfig.mqttUser;
  doc["mqtt_password"] = gConfig.mqttPassword;
  doc["mqtt_topic"] = gConfig.mqttTopic;
  doc["http_path"] = gConfig.httpPath;
  doc["device_id"] = gConfig.deviceId;
  doc["device_token"] = gConfig.deviceToken;
  doc["telemetry_interval_ms"] = gConfig.telemetryIntervalMs;
  doc["enable_analog"] = gConfig.enableAnalog;
  doc["dht_pin"] = gConfig.dhtPin;
  doc["dht_type"] = gConfig.dhtType;
  doc["onewire_pin"] = gConfig.oneWirePin;
  doc["i2c_sda"] = gConfig.i2cSda;
  doc["i2c_scl"] = gConfig.i2cScl;

  File file = LittleFS.open(CONFIG_PATH, "w");
  if (!file) {
    gRuntime.lastError = "failed to open config for write";
    return false;
  }

  const size_t written = serializeJsonPretty(doc, file);
  file.close();
  if (written == 0) {
    gRuntime.lastError = "failed to write config";
    return false;
  }

  logLine("config saved");
  return true;
}

void eraseStoredConfig() {
  if (LittleFS.exists(CONFIG_PATH)) {
    LittleFS.remove(CONFIG_PATH);
  }
  WiFi.disconnect(true);
  delay(100);
}

void onPortalSave() {
  gShouldSaveConfig = true;
}

unsigned long parseUnsigned(const char *text, unsigned long fallback) {
  if (text == nullptr || text[0] == '\0') {
    return fallback;
  }
  char *end = nullptr;
  const unsigned long value = strtoul(text, &end, 10);
  if (end == text) {
    return fallback;
  }
  return value;
}

void launchConfigPortal(bool forcePortal) {
  gShouldSaveConfig = false;
  char protocolValue[16];
  char hostValue[64];
  char tcpPortValue[8];
  char httpPortValue[8];
  char mqttPortValue[8];
  char mqttUserValue[32];
  char mqttPasswordValue[32];
  char mqttTopicValue[96];
  char httpPathValue[96];
  char deviceIdValue[40];
  char deviceTokenValue[48];
  char intervalValue[12];
  char dhtPinValue[8];
  char dhtTypeValue[12];
  char oneWirePinValue[8];

  snprintf(protocolValue, sizeof(protocolValue), "%s", gConfig.protocol.c_str());
  snprintf(hostValue, sizeof(hostValue), "%s", gConfig.serverHost.c_str());
  snprintf(tcpPortValue, sizeof(tcpPortValue), "%u", gConfig.tcpPort);
  snprintf(httpPortValue, sizeof(httpPortValue), "%u", gConfig.httpPort);
  snprintf(mqttPortValue, sizeof(mqttPortValue), "%u", gConfig.mqttPort);
  snprintf(mqttUserValue, sizeof(mqttUserValue), "%s", gConfig.mqttUser.c_str());
  snprintf(mqttPasswordValue, sizeof(mqttPasswordValue), "%s", gConfig.mqttPassword.c_str());
  snprintf(mqttTopicValue, sizeof(mqttTopicValue), "%s", gConfig.mqttTopic.c_str());
  snprintf(httpPathValue, sizeof(httpPathValue), "%s", gConfig.httpPath.c_str());
  snprintf(deviceIdValue, sizeof(deviceIdValue), "%s", gConfig.deviceId.c_str());
  snprintf(deviceTokenValue, sizeof(deviceTokenValue), "%s", gConfig.deviceToken.c_str());
  snprintf(intervalValue, sizeof(intervalValue), "%lu", gConfig.telemetryIntervalMs);
  snprintf(dhtPinValue, sizeof(dhtPinValue), "%u", gConfig.dhtPin);
  snprintf(dhtTypeValue, sizeof(dhtTypeValue), "%s", gConfig.dhtType.c_str());
  snprintf(oneWirePinValue, sizeof(oneWirePinValue), "%u", gConfig.oneWirePin);

  WiFiManager manager;
  manager.setSaveConfigCallback(onPortalSave);
  manager.setConfigPortalTimeout(180);
  manager.setConnectTimeout(30);
  manager.setBreakAfterConfig(true);

  WiFiManagerParameter protocolParam("protocol", "Protocol tcp/http/mqtt", protocolValue, sizeof(protocolValue));
  WiFiManagerParameter hostParam("server_host", "Server Host", hostValue, sizeof(hostValue));
  WiFiManagerParameter tcpPortParam("tcp_port", "TCP Port", tcpPortValue, sizeof(tcpPortValue));
  WiFiManagerParameter httpPortParam("http_port", "HTTP Port", httpPortValue, sizeof(httpPortValue));
  WiFiManagerParameter mqttPortParam("mqtt_port", "MQTT Port", mqttPortValue, sizeof(mqttPortValue));
  WiFiManagerParameter mqttUserParam("mqtt_user", "MQTT User", mqttUserValue, sizeof(mqttUserValue));
  WiFiManagerParameter mqttPasswordParam("mqtt_password", "MQTT Password", mqttPasswordValue, sizeof(mqttPasswordValue));
  WiFiManagerParameter mqttTopicParam("mqtt_topic", "MQTT Topic", mqttTopicValue, sizeof(mqttTopicValue));
  WiFiManagerParameter httpPathParam("http_path", "HTTP Path", httpPathValue, sizeof(httpPathValue));
  WiFiManagerParameter deviceIdParam("device_id", "Device ID", deviceIdValue, sizeof(deviceIdValue));
  WiFiManagerParameter deviceTokenParam("device_token", "Device Token", deviceTokenValue, sizeof(deviceTokenValue));
  WiFiManagerParameter intervalParam("telemetry_interval_ms", "Telemetry Interval ms", intervalValue, sizeof(intervalValue));
  WiFiManagerParameter dhtPinParam("dht_pin", "DHT Pin", dhtPinValue, sizeof(dhtPinValue));
  WiFiManagerParameter dhtTypeParam("dht_type", "DHT Type none/dht11/dht22", dhtTypeValue, sizeof(dhtTypeValue));
  WiFiManagerParameter oneWirePinParam("onewire_pin", "DS18B20 Pin", oneWirePinValue, sizeof(oneWirePinValue));

  manager.addParameter(&protocolParam);
  manager.addParameter(&hostParam);
  manager.addParameter(&tcpPortParam);
  manager.addParameter(&httpPortParam);
  manager.addParameter(&mqttPortParam);
  manager.addParameter(&mqttUserParam);
  manager.addParameter(&mqttPasswordParam);
  manager.addParameter(&mqttTopicParam);
  manager.addParameter(&httpPathParam);
  manager.addParameter(&deviceIdParam);
  manager.addParameter(&deviceTokenParam);
  manager.addParameter(&intervalParam);
  manager.addParameter(&dhtPinParam);
  manager.addParameter(&dhtTypeParam);
  manager.addParameter(&oneWirePinParam);

  const String apName = "MVP-ESP8266-" + chipIdHex();
  bool connected = false;
  if (forcePortal) {
    logLine("starting forced config portal");
    connected = manager.startConfigPortal(apName.c_str(), PORTAL_PASSWORD);
  } else {
    connected = manager.autoConnect(apName.c_str(), PORTAL_PASSWORD);
  }

  gConfig.protocol = protocolParam.getValue();
  gConfig.serverHost = hostParam.getValue();
  gConfig.tcpPort = normalizedPort(parseUnsigned(tcpPortParam.getValue(), gConfig.tcpPort), DEFAULT_TCP_PORT);
  gConfig.httpPort = normalizedPort(parseUnsigned(httpPortParam.getValue(), gConfig.httpPort), DEFAULT_HTTP_PORT);
  gConfig.mqttPort = normalizedPort(parseUnsigned(mqttPortParam.getValue(), gConfig.mqttPort), DEFAULT_MQTT_PORT);
  gConfig.mqttUser = mqttUserParam.getValue();
  gConfig.mqttPassword = mqttPasswordParam.getValue();
  gConfig.mqttTopic = mqttTopicParam.getValue();
  gConfig.httpPath = httpPathParam.getValue();
  gConfig.deviceId = deviceIdParam.getValue();
  gConfig.deviceToken = deviceTokenParam.getValue();
  gConfig.telemetryIntervalMs = normalizedInterval(parseUnsigned(intervalParam.getValue(), gConfig.telemetryIntervalMs));
  gConfig.dhtPin = normalizedPin(parseUnsigned(dhtPinParam.getValue(), gConfig.dhtPin), DEFAULT_DHT_PIN);
  gConfig.dhtType = dhtTypeParam.getValue();
  gConfig.oneWirePin = normalizedPin(parseUnsigned(oneWirePinParam.getValue(), gConfig.oneWirePin), DEFAULT_ONEWIRE_PIN);

  gConfig.protocol.trim();
  gConfig.serverHost.trim();
  gConfig.mqttTopic.trim();
  gConfig.httpPath.trim();
  gConfig.deviceId.trim();
  gConfig.deviceToken.trim();
  gConfig.dhtType.trim();
  if (gConfig.deviceId.isEmpty()) {
    gConfig.deviceId = fallbackDeviceId();
  }
  if (gConfig.mqttTopic.isEmpty()) {
    gConfig.mqttTopic = defaultMqttTopic();
  }
  if (gConfig.httpPath.isEmpty()) {
    gConfig.httpPath = DEFAULT_HTTP_PATH;
  }
  if (gConfig.protocol.isEmpty()) {
    gConfig.protocol = "tcp";
  }

  if (gShouldSaveConfig || forcePortal) {
    saveConfig();
  }

  if (!connected) {
    gRuntime.lastError = "wifi manager failed or portal timed out";
  }
}

void freeSensorDrivers() {
  if (gDallas != nullptr) {
    delete gDallas;
    gDallas = nullptr;
  }
  if (gOneWire != nullptr) {
    delete gOneWire;
    gOneWire = nullptr;
  }
  if (gDht != nullptr) {
    delete gDht;
    gDht = nullptr;
  }
  gSensors = SensorState{};
}

void initializeSensors() {
  freeSensorDrivers();

  Wire.begin(gConfig.i2cSda, gConfig.i2cScl);
  gSensors.bme280Active = gBme280.begin(0x76) || gBme280.begin(0x77);
  gSensors.bh1750Active = gBh1750.begin();

  gOneWire = new OneWire(gConfig.oneWirePin);
  gDallas = new DallasTemperature(gOneWire);
  gDallas->begin();
  gSensors.ds18b20Count = static_cast<uint8_t>(gDallas->getDeviceCount());
  gSensors.ds18b20Active = gSensors.ds18b20Count > 0;

  const String dhtType = lowerCopy(gConfig.dhtType);
  if (dhtType == "dht11" || dhtType == "dht22") {
    gDht = new DHT(gConfig.dhtPin, dhtType == "dht11" ? DHT11 : DHT22);
    gDht->begin();
    const float sampleHumidity = gDht->readHumidity();
    const float sampleTemp = gDht->readTemperature();
    gSensors.dhtActive = !isnan(sampleHumidity) || !isnan(sampleTemp);
  }

  logLine("sensor init summary: bme280=" + String(gSensors.bme280Active ? "on" : "off") +
    " bh1750=" + String(gSensors.bh1750Active ? "on" : "off") +
    " ds18b20=" + String(gSensors.ds18b20Active ? "on" : "off") +
    " dht=" + String(gSensors.dhtActive ? "on" : "off"));
}

void collectTelemetry(JsonObject values) {
  values["chip_id"] = chipIdHex();
  values["fw_version"] = FW_VERSION;
  values["rssi"] = WiFi.RSSI();
  values["free_heap"] = ESP.getFreeHeap();
  values["uptime_sec"] = millis() / 1000UL;

  if (gSensors.bme280Active) {
    const float temperature = gBme280.readTemperature();
    const float humidity = gBme280.readHumidity();
    const float pressure = gBme280.readPressure() / 100.0F;
    if (!isnan(temperature)) {
      values["temperature"] = temperature;
    }
    if (!isnan(humidity)) {
      values["humidity"] = humidity;
    }
    if (!isnan(pressure)) {
      values["pressure"] = pressure;
    }
  }

  if (gSensors.bh1750Active) {
    const float lux = gBh1750.readLightLevel();
    if (lux >= 0 && lux < 70000) {
      values["illuminance"] = lux;
    }
  }

  if (gSensors.ds18b20Active && gDallas != nullptr) {
    gDallas->requestTemperatures();
    const float probeTemp = gDallas->getTempCByIndex(0);
    if (probeTemp != DEVICE_DISCONNECTED_C) {
      if (!values.containsKey("temperature")) {
        values["temperature"] = probeTemp;
      } else {
        values["probe_temperature"] = probeTemp;
      }
    }
  }

  if (gSensors.dhtActive && gDht != nullptr) {
    const float dhtTemp = gDht->readTemperature();
    const float dhtHumidity = gDht->readHumidity();
    if (!isnan(dhtTemp)) {
      if (!values.containsKey("temperature")) {
        values["temperature"] = dhtTemp;
      } else {
        values["dht_temperature"] = dhtTemp;
      }
    }
    if (!isnan(dhtHumidity)) {
      if (!values.containsKey("humidity")) {
        values["humidity"] = dhtHumidity;
      } else {
        values["dht_humidity"] = dhtHumidity;
      }
    }
  }

  if (gConfig.enableAnalog) {
    values["analog"] = analogRead(A0);
  }
}

bool buildPayloadForTransport(TransportMode mode, String &payload) {
  DynamicJsonDocument doc(MEDIUM_JSON_BYTES);
  if (mode == TransportMode::TCP) {
    doc["type"] = "telemetry";
  } else {
    doc["device_id"] = gConfig.deviceId;
    doc["token"] = gConfig.deviceToken;
    doc["protocol"] = mode == TransportMode::MQTT ? "mqtt_json" : "http_json";
  }

  JsonObject values = doc.createNestedObject("values");
  collectTelemetry(values);
  payload = "";
  serializeJson(doc, payload);
  return payload.length() > 0;
}

bool writeTcpJsonLine(const String &payload) {
  if (!gTcpClient.connected()) {
    return false;
  }
  const size_t written = gTcpClient.print(payload);
  gTcpClient.print('\n');
  return written == payload.length();
}

void sendTcpAuth() {
  DynamicJsonDocument doc(SMALL_JSON_BYTES);
  doc["type"] = "auth";
  doc["device_id"] = gConfig.deviceId;
  doc["token"] = gConfig.deviceToken;
  String payload;
  serializeJson(doc, payload);
  writeTcpJsonLine(payload);
}

void sendAck(const String &commandId, const String &status, const String &message) {
  DynamicJsonDocument doc(SMALL_JSON_BYTES);
  doc["command_id"] = commandId;
  doc["status"] = status;
  doc["message"] = message;

  if (currentTransport() == TransportMode::TCP) {
    doc["type"] = "ack";
    String payload;
    serializeJson(doc, payload);
    if (writeTcpJsonLine(payload)) {
      gRuntime.lastAckStatus = status;
    }
    return;
  }

  if (currentTransport() == TransportMode::MQTT && gMqttClient.connected()) {
    const String topic = mqttAckTopic();
    String payload;
    serializeJson(doc, payload);
    if (gMqttClient.publish(topic.c_str(), payload.c_str(), false)) {
      gRuntime.lastAckStatus = status;
    }
  }
}

void scheduleRestart() {
  gRuntime.restartRequested = true;
  gRuntime.restartAtMs = millis() + RESTART_DELAY_MS;
}

void handleCommandDocument(JsonObjectConst root) {
  const String commandId = root["command_id"] | (String("local-") + String(millis()));
  const String name = lowerCopy(root["name"] | "");
  gRuntime.lastCommand = name;

  if (name == "reboot" || name == "restart") {
    sendAck(commandId, "ok", "restart scheduled");
    scheduleRestart();
    return;
  }

  if (name == "telemetry_once") {
    gRuntime.forceTelemetry = true;
    sendAck(commandId, "ok", "telemetry scheduled");
    return;
  }

  if (name == "set_interval") {
    JsonVariantConst params = root["params"];
    unsigned long interval = params["ms"] | 0UL;
    if (interval == 0UL) {
      interval = params["interval_ms"] | gConfig.telemetryIntervalMs;
    }
    interval = normalizedInterval(interval);
    gConfig.telemetryIntervalMs = interval;
    saveConfig();
    sendAck(commandId, "ok", "interval updated");
    return;
  }

  if (name == "reset_config") {
    sendAck(commandId, "ok", "config reset scheduled");
    eraseStoredConfig();
    scheduleRestart();
    return;
  }

  sendAck(commandId, "ok", "command accepted");
}

void handleTcpFrame(const String &line) {
  DynamicJsonDocument doc(SMALL_JSON_BYTES);
  DeserializationError error = deserializeJson(doc, line);
  if (error) {
    gRuntime.lastError = String("invalid tcp frame: ") + error.c_str();
    return;
  }

  const String type = lowerCopy(doc["type"] | "");
  if (type == "auth_ok") {
    gRuntime.tcpAuthenticated = true;
    logLine("tcp session authenticated");
    return;
  }
  if (type == "pong") {
    return;
  }
  if (type == "command") {
    handleCommandDocument(doc.as<JsonObjectConst>());
    return;
  }
  if (type == "error") {
    gRuntime.lastError = doc["message"] | "server returned error";
  }
}

void pollTcpClient() {
  if (!gTcpClient.connected()) {
    gRuntime.tcpAuthenticated = false;
    return;
  }

  while (gTcpClient.available() > 0) {
    const char ch = static_cast<char>(gTcpClient.read());
    if (ch == '\n') {
      gTcpReadBuffer.trim();
      if (!gTcpReadBuffer.isEmpty()) {
        handleTcpFrame(gTcpReadBuffer);
      }
      gTcpReadBuffer = "";
    } else if (ch != '\r') {
      if (gTcpReadBuffer.length() < 1024) {
        gTcpReadBuffer += ch;
      } else {
        gTcpReadBuffer = "";
      }
    }
  }
}

void ensureTcpConnection() {
  if (gTcpClient.connected()) {
    return;
  }
  if (millis() - gRuntime.lastConnectAttemptAtMs < RETRY_BACKOFF_MS) {
    return;
  }

  gRuntime.lastConnectAttemptAtMs = millis();
  gRuntime.tcpAuthenticated = false;
  gTcpReadBuffer = "";
  gTcpClient.stop();

  logLine("connecting tcp " + gConfig.serverHost + ":" + String(gConfig.tcpPort));
  if (!gTcpClient.connect(gConfig.serverHost.c_str(), gConfig.tcpPort)) {
    gRuntime.lastError = "tcp connect failed";
    return;
  }

  gTcpClient.setNoDelay(true);
  sendTcpAuth();
}

void sendTcpPingIfDue() {
  if (!gTcpClient.connected() || !gRuntime.tcpAuthenticated) {
    return;
  }
  if (millis() - gRuntime.lastTcpPingAtMs < TCP_PING_INTERVAL_MS) {
    return;
  }

  DynamicJsonDocument doc(SMALL_JSON_BYTES);
  doc["type"] = "ping";
  String payload;
  serializeJson(doc, payload);
  if (writeTcpJsonLine(payload)) {
    gRuntime.lastTcpPingAtMs = millis();
  }
}

bool sendHttpTelemetry(const String &payload) {
  if (WiFi.status() != WL_CONNECTED) {
    return false;
  }

  HTTPClient http;
  const String url = "http://" + gConfig.serverHost + ":" + String(gConfig.httpPort) + httpIngestPath();
  if (!http.begin(gHttpClient, url)) {
    gRuntime.lastError = "http begin failed";
    return false;
  }

  http.addHeader("Content-Type", "application/json");
  const int statusCode = http.POST(payload);
  http.end();
  if (statusCode < 200 || statusCode >= 300) {
    gRuntime.lastError = "http post failed: " + String(statusCode);
    return false;
  }
  return true;
}

bool mqttConnectWithCredentials(const String &clientId) {
  if (gConfig.mqttUser.length() > 0) {
    return gMqttClient.connect(clientId.c_str(), gConfig.mqttUser.c_str(), gConfig.mqttPassword.c_str());
  }
  if (gConfig.deviceId.length() > 0 && gConfig.deviceToken.length() > 0) {
    return gMqttClient.connect(clientId.c_str(), gConfig.deviceId.c_str(), gConfig.deviceToken.c_str());
  }
  return gMqttClient.connect(clientId.c_str());
}

void handleMqttPayload(char *topic, byte *payload, unsigned int length) {
  String body;
  body.reserve(length);
  for (unsigned int index = 0; index < length; ++index) {
    body += static_cast<char>(payload[index]);
  }

  DynamicJsonDocument doc(SMALL_JSON_BYTES);
  DeserializationError error = deserializeJson(doc, body);
  if (error) {
    gRuntime.lastError = String("invalid mqtt payload on ") + topic;
    return;
  }
  handleCommandDocument(doc.as<JsonObjectConst>());
}

void ensureMqttConnection() {
  if (gMqttClient.connected()) {
    return;
  }
  if (millis() - gRuntime.lastConnectAttemptAtMs < RETRY_BACKOFF_MS) {
    return;
  }

  gRuntime.lastConnectAttemptAtMs = millis();
  gMqttClient.setServer(gConfig.serverHost.c_str(), gConfig.mqttPort);
  gMqttClient.setBufferSize(1024);
  gMqttClient.setKeepAlive(30);
  gMqttClient.setCallback(handleMqttPayload);

  const String clientId = gConfig.deviceId + "-" + chipIdHex();
  logLine("connecting mqtt " + gConfig.serverHost + ":" + String(gConfig.mqttPort));
  if (!mqttConnectWithCredentials(clientId)) {
    gRuntime.lastError = "mqtt connect failed, state=" + String(gMqttClient.state());
    return;
  }

  const String downTopic = mqttDownTopic();
  gMqttClient.subscribe(downTopic.c_str());
  logLine("mqtt connected, subscribed " + downTopic);
}

bool sendMqttTelemetry(const String &payload) {
  ensureMqttConnection();
  if (!gMqttClient.connected()) {
    return false;
  }
  const String topic = mqttUpTopic();
  if (!gMqttClient.publish(topic.c_str(), payload.c_str(), false)) {
    gRuntime.lastError = "mqtt publish failed";
    return false;
  }
  return true;
}

bool sendTelemetry() {
  String payload;
  const TransportMode mode = currentTransport();
  if (!buildPayloadForTransport(mode, payload)) {
    return false;
  }

  bool sent = false;
  if (mode == TransportMode::TCP) {
    ensureTcpConnection();
    pollTcpClient();
    if (gTcpClient.connected() && gRuntime.tcpAuthenticated) {
      sent = writeTcpJsonLine(payload);
    }
  } else if (mode == TransportMode::HTTP) {
    sent = sendHttpTelemetry(payload);
  } else if (mode == TransportMode::MQTT) {
    sent = sendMqttTelemetry(payload);
  }

  if (sent) {
    gRuntime.lastTelemetryAtMs = millis();
    gRuntime.lastError = "";
    logLine("telemetry sent via " + lowerCopy(gConfig.protocol));
  }
  return sent;
}

void sendStatusPage() {
  String html;
  html.reserve(2200);
  html += F("<!doctype html><html><head><meta charset='utf-8'><meta name='viewport' content='width=device-width,initial-scale=1'>");
  html += F("<title>MVP ESP8266</title><style>body{font-family:Arial,sans-serif;background:#f5f7fb;color:#122033;padding:24px;line-height:1.5}main{max-width:860px;margin:0 auto;background:#fff;border-radius:18px;padding:24px;box-shadow:0 14px 40px rgba(12,26,75,.12)}code{background:#eef2ff;padding:2px 6px;border-radius:8px}a{color:#1459d9}table{width:100%;border-collapse:collapse;margin-top:16px}td{padding:8px 10px;border-bottom:1px solid #e5ebf5}h1{margin-top:0}.pill{display:inline-block;padding:4px 10px;border-radius:999px;background:#eef5ff;margin-right:8px}</style></head><body><main>");
  html += "<h1>MVP ESP8266 Universal Agent</h1>";
  html += "<p><span class='pill'>" + lowerCopy(gConfig.protocol) + "</span><span class='pill'>" + WiFi.localIP().toString() + "</span><span class='pill'>FW " FW_VERSION "</span></p>";
  html += "<p><a href='/status'>/status</a> | <a href='/update'>/update</a> | <a href='/reboot'>/reboot</a> | <a href='/reset-config'>/reset-config</a></p>";
  html += "<table>";
  html += "<tr><td>Hostname</td><td><code>" + gHostname + "</code></td></tr>";
  html += "<tr><td>Device ID</td><td><code>" + gConfig.deviceId + "</code></td></tr>";
  html += "<tr><td>Token</td><td><code>" + maskedSecret(gConfig.deviceToken) + "</code></td></tr>";
  html += "<tr><td>Server</td><td><code>" + gConfig.serverHost + "</code></td></tr>";
  html += "<tr><td>TCP</td><td><code>" + String(gConfig.tcpPort) + "</code></td></tr>";
  html += "<tr><td>HTTP</td><td><code>" + httpIngestPath() + "</code></td></tr>";
  html += "<tr><td>MQTT Topic</td><td><code>" + mqttUpTopic() + "</code></td></tr>";
  html += "<tr><td>Interval</td><td><code>" + String(gConfig.telemetryIntervalMs) + " ms</code></td></tr>";
  html += "<tr><td>Last Error</td><td><code>" + (gRuntime.lastError.length() ? gRuntime.lastError : "none") + "</code></td></tr>";
  html += "<tr><td>Sensors</td><td><code>BME280=" + String(gSensors.bme280Active ? "on" : "off") +
    " BH1750=" + String(gSensors.bh1750Active ? "on" : "off") +
    " DS18B20=" + String(gSensors.ds18b20Active ? "on" : "off") +
    " DHT=" + String(gSensors.dhtActive ? "on" : "off") + "</code></td></tr>";
  html += "</table></main></body></html>";
  gAdminServer.send(200, "text/html; charset=utf-8", html);
}

void sendStatusJson() {
  DynamicJsonDocument doc(MEDIUM_JSON_BYTES);
  doc["firmware"] = FW_VERSION;
  doc["hostname"] = gHostname;
  doc["chip_id"] = chipIdHex();
  doc["ip"] = WiFi.localIP().toString();
  doc["mac"] = WiFi.macAddress();
  doc["ssid"] = WiFi.SSID();
  doc["protocol"] = lowerCopy(gConfig.protocol);
  doc["server_host"] = gConfig.serverHost;
  doc["tcp_port"] = gConfig.tcpPort;
  doc["http_port"] = gConfig.httpPort;
  doc["mqtt_port"] = gConfig.mqttPort;
  doc["mqtt_topic"] = mqttUpTopic();
  doc["http_path"] = httpIngestPath();
  doc["device_id"] = gConfig.deviceId;
  doc["token_masked"] = maskedSecret(gConfig.deviceToken);
  doc["telemetry_interval_ms"] = gConfig.telemetryIntervalMs;
  doc["last_error"] = gRuntime.lastError;
  doc["last_command"] = gRuntime.lastCommand;
  doc["last_ack_status"] = gRuntime.lastAckStatus;
  doc["tcp_authenticated"] = gRuntime.tcpAuthenticated;
  doc["uptime_sec"] = millis() / 1000UL;

  JsonObject sensors = doc.createNestedObject("sensors");
  sensors["bme280"] = gSensors.bme280Active;
  sensors["bh1750"] = gSensors.bh1750Active;
  sensors["ds18b20"] = gSensors.ds18b20Active;
  sensors["ds18b20_count"] = gSensors.ds18b20Count;
  sensors["dht"] = gSensors.dhtActive;
  sensors["dht_type"] = gConfig.dhtType;
  sensors["analog"] = gConfig.enableAnalog;

  String body;
  serializeJsonPretty(doc, body);
  gAdminServer.send(200, "application/json; charset=utf-8", body);
}

void startAdminServer() {
  if (gRuntime.adminServerStarted) {
    return;
  }

  gAdminServer.on("/", HTTP_GET, sendStatusPage);
  gAdminServer.on("/status", HTTP_GET, sendStatusJson);
  gAdminServer.on("/reboot", HTTP_ANY, []() {
    gAdminServer.send(200, "text/plain; charset=utf-8", "reboot scheduled");
    scheduleRestart();
  });
  gAdminServer.on("/reset-config", HTTP_ANY, []() {
    gAdminServer.send(200, "text/plain; charset=utf-8", "config reset scheduled; portal will open on next boot");
    eraseStoredConfig();
    scheduleRestart();
  });

  gUpdaterPassword = gConfig.deviceToken.length() ? gConfig.deviceToken : String("admin12345");
  gHttpUpdater.setup(&gAdminServer, "/update", "admin", gUpdaterPassword.c_str());
  gAdminServer.begin();
  gRuntime.adminServerStarted = true;

  if (MDNS.begin(gHostname.c_str())) {
    MDNS.addService("http", "tcp", 80);
    gRuntime.mdnsReady = true;
  }
}

void maintainTransport() {
  if (currentTransport() == TransportMode::TCP) {
    if (gMqttClient.connected()) {
      gMqttClient.disconnect();
    }
    ensureTcpConnection();
    pollTcpClient();
    sendTcpPingIfDue();
    return;
  }

  if (currentTransport() == TransportMode::MQTT) {
    if (gTcpClient.connected()) {
      gTcpClient.stop();
      gRuntime.tcpAuthenticated = false;
    }
    ensureMqttConnection();
    if (gMqttClient.connected()) {
      gMqttClient.loop();
    }
    return;
  }

  if (gMqttClient.connected()) {
    gMqttClient.disconnect();
  }
  if (gTcpClient.connected()) {
    gTcpClient.stop();
    gRuntime.tcpAuthenticated = false;
  }
}

}  // namespace

void setup() {
  Serial.begin(115200);
  Serial.println();
  logLine("booting firmware " FW_VERSION);

  if (!LittleFS.begin()) {
    logLine("LittleFS mount failed; continuing without persisted config");
  }

  gHostname = defaultHostname();
  loadConfig();

  WiFi.persistent(false);
  WiFi.mode(WIFI_STA);
  WiFi.hostname(gHostname.c_str());
  launchConfigPortal(!isConfigValid());

  if (WiFi.status() != WL_CONNECTED) {
    logLine("wifi not connected after setup");
  } else {
    logLine("wifi connected, ip=" + WiFi.localIP().toString());
  }

  initializeSensors();
  startAdminServer();

  gMqttClient.setBufferSize(1024);
  gMqttClient.setKeepAlive(30);
  gRuntime.forceTelemetry = true;
}

void loop() {
  gAdminServer.handleClient();
  if (gRuntime.mdnsReady) {
    MDNS.update();
  }

  if (WiFi.status() == WL_CONNECTED) {
    maintainTransport();

    if (gRuntime.forceTelemetry || millis() - gRuntime.lastTelemetryAtMs >= gConfig.telemetryIntervalMs) {
      if (sendTelemetry()) {
        gRuntime.forceTelemetry = false;
      }
    }
  }

  if (gRuntime.restartRequested && millis() >= gRuntime.restartAtMs) {
    ESP.restart();
  }

  delay(10);
}
