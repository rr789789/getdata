const VIEW_TITLE_KEYS = {
  overview: "nav_overview",
  products: "nav_products",
  devices: "nav_devices",
  governance: "nav_governance",
  config: "nav_config",
  simulator: "nav_simulator",
};

const runtimeConfig = resolveRuntimeConfig();
const API_BASE_URL = normalizeBaseURL(runtimeConfig.api_base_url || "");

function resolveRuntimeConfig() {
  const config = window.__MVP_RUNTIME_CONFIG__ || {};
  return typeof config === "object" && config !== null ? config : {};
}

function normalizeBaseURL(value) {
  const raw = String(value || "").trim();
  if (!raw) {
    return "";
  }
  return raw.replace(/\/+$/, "");
}

function resolveAPIURL(path) {
  const raw = String(path || "").trim();
  if (!raw) {
    return raw;
  }
  if (/^[a-z]+:\/\//i.test(raw)) {
    return raw;
  }
  if (!API_BASE_URL) {
    return raw;
  }
  return raw.startsWith("/") ? `${API_BASE_URL}${raw}` : `${API_BASE_URL}/${raw}`;
}

function buildHTTPIngestPath(deviceID) {
  return resolveAPIURL(`/api/v1/ingest/http/${encodeURIComponent(deviceID)}`);
}

function runtimeModeLabel() {
  const mode = runtimeConfig.desktop_mode
    ? (appState.locale === "zh" ? "桌面端" : "Desktop")
    : (appState.locale === "zh" ? "Web后台" : "Web Admin");
  const target = API_BASE_URL || (appState.locale === "zh" ? "同源 API" : "Embedded API");
  return `${mode} | ${target}`;
}

async function ensureInstalled() {
  const status = await requestJSON("/api/v1/install/status");
  if (status && status.installed === false) {
    window.location.replace("/install");
    return false;
  }
  return true;
}

const I18N = {
  zh: {
    app_title: "MVP IoT 控制台",
    locale_button: "中文 / EN",
    nav_overview: "总览",
    nav_products: "产品中心",
    nav_devices: "设备中心",
    nav_governance: "治理中心",
    nav_config: "配置中心",
    nav_simulator: "模拟器实验室",
    refresh: "刷新",
    runtime_loading: "正在加载运行状态",
    runtime_healthy: "运行正常 | {time}",
    runtime_unavailable: "运行状态不可用",
    products_metric: "产品",
    devices_metric: "设备",
    online_metric: "在线",
    groups_metric: "分组",
    rules_metric: "规则",
    alerts_metric: "告警",
    configs_metric: "配置",
    connections_metric: "连接",
    telemetry_metric: "遥测",
    command_ack_metric: "命令回执",
    http_ingest_metric: "HTTP 接入成功",
    mqtt_metric: "MQTT 消息",
    samples_metric: "遥测样本",
    persist_errors_metric: "持久化错误",
    no_products: "暂无产品。",
    no_devices: "暂无设备。",
    no_groups: "暂无分组。",
    no_rules: "暂无规则。",
    no_alerts: "暂无告警。",
    no_configs: "暂无配置模板。",
    no_simulators: "暂无模拟器。",
    no_tags: "暂无标签",
    no_metadata: "暂无元数据",
    unbound: "未绑定",
    any_product: "任意产品",
    optional: "可选",
    auto: "自动",
    online: "在线",
    offline: "离线",
    inspect_device: "查看设备",
    remove_selected_device: "移出当前设备",
    add_selected_device: "加入当前设备",
    select_device_membership: "先选择一个设备，再管理分组成员。",
    status: "状态",
    updated: "更新时间",
    all_products: "全部产品",
    select_device_apply: "先选择一个设备再下发。",
    product_scope_mismatch: "当前设备产品与该模板不匹配。",
    apply_selected_device: "应用到当前设备",
    selected_device_none: "未选择",
    selected_device_empty: "选择一个设备以查看标签、影子、命令和告警。",
    command_accepted: "命令已接受",
    created_ok: "创建成功",
    shadow_updated: "影子已更新",
    tags_updated: "标签已更新",
    config_applied: "配置已下发",
    select_profile_first: "请先选择配置模板",
    processing_note: "处理备注",
    create_in_progress: "创建中...",
    update_in_progress: "更新中...",
    send_in_progress: "发送中...",
    apply_in_progress: "应用中...",
    acknowledged: "已确认",
    resolved: "已关闭",
    latest_alerts: "最新告警",
    recent_devices: "最近设备",
    protocol_templates_loading: "正在加载协议模板...",
    protocol_templates_empty: "暂无协议模板。",
    protocol_templates_title: "协议与传感器模板",
    protocol_templates_desc: "常见传感器协议模板可直接作为产品接入配置起点。",
    apply_template: "应用模板",
    connect: "连接",
    disconnect: "断开",
    remove: "移除",
    send_telemetry: "发送遥测",
    protocol_template: "协议模板",
    sensor_template: "传感器模板",
    common_sensors: "常见传感器",
    example_payload: "示例载荷",
    point_mappings: "点位映射",
    transport: "传输",
    protocol: "协议",
    ingest_mode: "接入方式",
    payload_format: "载荷格式",
    auth_mode: "鉴权方式",
    topic_path: "主题 / 路径",
    no_logs: "暂无日志。",
  },
  en: {
    app_title: "MVP IoT Console",
    locale_button: "中文 / EN",
    nav_overview: "Overview",
    nav_products: "Product Center",
    nav_devices: "Device Center",
    nav_governance: "Governance",
    nav_config: "Config Center",
    nav_simulator: "Simulator Lab",
    refresh: "Refresh",
    runtime_loading: "Loading runtime status",
    runtime_healthy: "Runtime healthy | {time}",
    runtime_unavailable: "Runtime unavailable",
    products_metric: "Products",
    devices_metric: "Devices",
    online_metric: "Online",
    groups_metric: "Groups",
    rules_metric: "Rules",
    alerts_metric: "Alerts",
    configs_metric: "Configs",
    connections_metric: "Connections",
    telemetry_metric: "Telemetry",
    command_ack_metric: "Command Ack",
    http_ingest_metric: "HTTP Ingest",
    mqtt_metric: "MQTT Messages",
    samples_metric: "Telemetry Samples",
    persist_errors_metric: "Persist Errors",
    no_products: "No products yet.",
    no_devices: "No devices yet.",
    no_groups: "No groups yet.",
    no_rules: "No rules yet.",
    no_alerts: "No alerts yet.",
    no_configs: "No config profiles yet.",
    no_simulators: "No simulators yet.",
    no_tags: "No tags",
    no_metadata: "No metadata",
    unbound: "Unbound",
    any_product: "Any product",
    optional: "Optional",
    auto: "Auto",
    online: "online",
    offline: "offline",
    inspect_device: "Inspect Device",
    remove_selected_device: "Remove selected device",
    add_selected_device: "Add selected device",
    select_device_membership: "Select a device to manage membership.",
    status: "Status",
    updated: "Updated",
    all_products: "All products",
    select_device_apply: "Select a device to apply.",
    product_scope_mismatch: "Selected device product does not match this profile.",
    apply_selected_device: "Apply To Selected Device",
    selected_device_none: "Unselected",
    selected_device_empty: "Select a device to inspect tags, shadow, commands and alerts.",
    command_accepted: "Command accepted",
    created_ok: "Created",
    shadow_updated: "Shadow updated",
    tags_updated: "Tags updated",
    config_applied: "Config applied",
    select_profile_first: "Select a config profile first",
    processing_note: "Processing note",
    create_in_progress: "Creating...",
    update_in_progress: "Updating...",
    send_in_progress: "Sending...",
    apply_in_progress: "Applying...",
    acknowledged: "Acknowledged",
    resolved: "Resolved",
    latest_alerts: "Latest Alerts",
    recent_devices: "Recent Devices",
    protocol_templates_loading: "Loading protocol catalog...",
    protocol_templates_empty: "No protocol templates.",
    protocol_templates_title: "Protocol And Sensor Templates",
    protocol_templates_desc: "Use common sensor protocol templates as a starting point for product access profiles.",
    apply_template: "Apply Template",
    connect: "Connect",
    disconnect: "Disconnect",
    remove: "Remove",
    send_telemetry: "Send Telemetry",
    protocol_template: "Protocol Template",
    sensor_template: "Sensor Template",
    common_sensors: "Common Sensors",
    example_payload: "Example Payload",
    point_mappings: "Point Mappings",
    transport: "Transport",
    protocol: "Protocol",
    ingest_mode: "Ingest Mode",
    payload_format: "Payload Format",
    auth_mode: "Auth Mode",
    topic_path: "Topic / Path",
    no_logs: "No logs yet.",
  },
};

function resolveInitialLocale() {
  const saved = window.localStorage.getItem("mvp_locale");
  if (saved === "zh" || saved === "en") {
    return saved;
  }
  return navigator.language.toLowerCase().startsWith("zh") ? "zh" : "en";
}

const appState = {
  locale: resolveInitialLocale(),
  currentView: "overview",
  health: null,
  metrics: null,
  systemInfo: null,
  products: [],
  devices: [],
  groups: [],
  rules: [],
  alerts: [],
  configProfiles: [],
  protocolCatalog: [],
  simulators: [],
  selectedDeviceId: "",
};

function t(key, variables = {}) {
  const localeTable = I18N[appState.locale] || I18N.en;
  const fallback = I18N.en[key] || key;
  let template = localeTable[key] || fallback;
  for (const [name, value] of Object.entries(variables)) {
    template = template.replaceAll(`{${name}}`, String(value));
  }
  return template;
}

async function requestJSON(path, options = {}) {
  const response = await fetch(resolveAPIURL(path), {
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
    ...options,
  });

  if (response.status === 204) {
    return null;
  }

  const contentType = response.headers.get("content-type") || "";
  const payload = contentType.includes("application/json") ? await response.json() : await response.text();
  if (!response.ok) {
    const message = typeof payload === "string" ? payload : (payload.error || response.statusText);
    throw new Error(message);
  }
  return payload;
}

function parseJSON(text, fallback) {
  const raw = String(text || "").trim();
  return raw ? JSON.parse(raw) : fallback;
}

function parseLooseValue(text) {
  const raw = String(text || "").trim();
  if (!raw) {
    return "";
  }
  if (raw === "true") {
    return true;
  }
  if (raw === "false") {
    return false;
  }
  if (!Number.isNaN(Number(raw))) {
    return Number(raw);
  }
  if (raw.startsWith("{") || raw.startsWith("[") || raw.startsWith("\"")) {
    return JSON.parse(raw);
  }
  return raw;
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll("\"", "&quot;");
}

function formatTime(value) {
  return value ? new Date(value).toLocaleString(appState.locale === "zh" ? "zh-CN" : "en-US") : "-";
}

function formatValue(value) {
  if (value === null || value === undefined) {
    return "-";
  }
  return typeof value === "object" ? JSON.stringify(value) : String(value);
}

function pretty(value) {
  try {
    return JSON.stringify(value ?? {}, null, 2);
  } catch (error) {
    return String(value ?? "");
  }
}

function isEditingTextField() {
  const element = document.activeElement;
  return !!element && (element.tagName === "INPUT" || element.tagName === "TEXTAREA");
}

function setHint(id, message, isError = false) {
  const node = document.getElementById(id);
  if (!node) {
    return;
  }
  node.textContent = message || "";
  node.style.color = isError ? "#b42318" : "";
}

function setText(selector, value) {
  const node = typeof selector === "string" ? document.querySelector(selector) : selector;
  if (node) {
    node.textContent = value;
  }
}

function setPlaceholder(id, value) {
  const node = document.getElementById(id);
  if (node) {
    node.placeholder = value;
  }
}

function applyTranslations() {
  document.documentElement.lang = appState.locale === "zh" ? "zh-CN" : "en";
  document.title = runtimeConfig.app_title || t("app_title");

  setText("#locale-toggle", t("locale_button"));
  setText("#refresh-button", t("refresh"));
  setText("#health-text", appState.health?.status === "ok" ? t("runtime_healthy", { time: formatTime(appState.health.time) }) : t("runtime_loading"));
  setText("#platform-pill", runtimeModeLabel());

  const viewTitle = document.getElementById("view-title");
  if (viewTitle) {
    viewTitle.textContent = t(VIEW_TITLE_KEYS[appState.currentView] || "nav_overview");
  }

  [
    ["[data-view-target=\"overview\"]", "nav_overview"],
    ["[data-view-target=\"products\"]", "nav_products"],
    ["[data-view-target=\"devices\"]", "nav_devices"],
    ["[data-view-target=\"governance\"]", "nav_governance"],
    ["[data-view-target=\"config\"]", "nav_config"],
    ["[data-view-target=\"simulator\"]", "nav_simulator"],
  ].forEach(([selector, key]) => setText(selector, t(key)));

  setText(".brand-block .subtle", appState.locale === "zh"
    ? "单二进制控制面，覆盖产品建模、设备接入、规则告警和模拟器联调。"
    : "Single binary control plane for product, fleet, rules and simulator workflows.");
  setText(".console-topbar .eyebrow", appState.locale === "zh" ? "设备云平台" : "Device Cloud Platform");
  setText(".hero-kicker", appState.locale === "zh" ? "企业级运维控制台" : "Enterprise operation console");
  setText(".hero-copy h3", appState.locale === "zh"
    ? "产品建模、设备治理、协议接入与模拟测试集中管理"
    : "Product modeling, fleet governance, protocol onboarding and simulator testing in one place");
  setText(".hero-copy .copy", appState.locale === "zh"
    ? "控制台参考企业物联网平台布局，补充了多协议产品接入配置、HTTP Push 接入和常见传感器协议模板。"
    : "This console follows an enterprise IoT control-plane style and now adds native TCP, HTTP and MQTT access profiles plus common sensor protocol templates.");
  setText(".hero-glance .glance-card:nth-of-type(1) .glance-label", appState.locale === "zh" ? "控制面" : "Control Plane");
  setText(".hero-glance .glance-card:nth-of-type(1) strong", appState.locale === "zh" ? "产品 / 设备 / 规则 / 协议" : "Product / Device / Rule / Access");
  setText(".hero-glance .glance-card:nth-of-type(1) small", appState.locale === "zh"
    ? "统一管理物模型、设备分组、告警生命周期、配置模板和接入协议。"
    : "Manage thing models, groups, alert lifecycle, config profiles and access protocols.");
  setText(".hero-glance .glance-card:nth-of-type(2) .glance-label", appState.locale === "zh" ? "接入链路" : "Gateway Path");
  setText(".hero-glance .glance-card:nth-of-type(2) strong", appState.locale === "zh" ? "TCP / HTTP Push / MQTT" : "TCP / HTTP Push / MQTT");
  setText(".hero-glance .glance-card:nth-of-type(2) small", appState.locale === "zh"
    ? "内置 TCP 网关继续可用，并新增 HTTP Push 统一接入端点承接 Modbus、MQTT、OPC UA 等桥接数据。"
    : "The built-in TCP gateway, HTTP ingest and MQTT broker can run together, while Modbus, OPC UA and BACnet still enter through bridge collectors.");
  setText(".panel-overview .section-kicker", appState.locale === "zh" ? "运行态" : "Runtime");
  setText(".panel-overview .section-head h3", appState.locale === "zh" ? "核心指标" : "Overview Metrics");
  const overviewDevicePanel = document.getElementById("overview-device-list")?.closest(".panel");
  const overviewAlertPanel = document.getElementById("overview-alert-list")?.closest(".panel");
  const protocolPanel = document.getElementById("protocol-catalog-list")?.closest(".panel");
  setText("#system-kicker", appState.locale === "zh" ? "系统" : "System");
  setText("#system-title", appState.locale === "zh" ? "实例概览" : "Instance Summary");
  setText(overviewDevicePanel?.querySelector(".section-kicker"), appState.locale === "zh" ? "设备队列" : "Fleet");
  setText(overviewDevicePanel?.querySelector(".section-head h3"), t("recent_devices"));
  setText(overviewAlertPanel?.querySelector(".section-kicker"), appState.locale === "zh" ? "告警" : "Alert");
  setText(overviewAlertPanel?.querySelector(".section-head h3"), t("latest_alerts"));
  setText("[data-view=\"products\"] .section-kicker", appState.locale === "zh" ? "产品" : "Product");
  setText("[data-view=\"products\"] .section-head h3", appState.locale === "zh" ? "产品与接入模型" : "Product And Access Model");
  setText(protocolPanel?.querySelector(".section-kicker"), appState.locale === "zh" ? "协议" : "Protocol");
  setText(protocolPanel?.querySelector(".section-head h3"), t("protocol_templates_title"));
  setText("#product-form label:nth-of-type(1) span", appState.locale === "zh" ? "产品名称" : "Product Name");
  setText("#product-form label:nth-of-type(2) span", appState.locale === "zh" ? "描述" : "Description");
  setText("#product-form label:nth-of-type(3) span", appState.locale === "zh" ? "传输" : "Transport");
  setText("#product-form label:nth-of-type(4) span", appState.locale === "zh" ? "协议" : "Protocol");
  setText("#product-form label:nth-of-type(5) span", appState.locale === "zh" ? "接入方式" : "Ingest Mode");
  setText("#product-form label:nth-of-type(6) span", appState.locale === "zh" ? "载荷格式" : "Payload Format");
  setText("#product-form label:nth-of-type(7) span", appState.locale === "zh" ? "传感器模板" : "Sensor Template");
  setText("#product-form label:nth-of-type(8) span", appState.locale === "zh" ? "鉴权方式" : "Auth Mode");
  setText("#product-form label:nth-of-type(9) span", appState.locale === "zh" ? "主题 / 路径" : "Topic / Path");
  setText("#product-form label:nth-of-type(10) span", appState.locale === "zh" ? "元数据 JSON" : "Metadata JSON");
  setText("#product-form label:nth-of-type(11) span", appState.locale === "zh" ? "点位映射 JSON" : "Point Mappings JSON");
  setText("#product-form label:nth-of-type(12) span", appState.locale === "zh" ? "物模型 JSON" : "Thing Model JSON");
  setText("#product-form button[type=\"submit\"]", appState.locale === "zh" ? "创建产品" : "Create Product");

  setText("[data-view=\"devices\"] .overview-grid .panel:nth-of-type(1) .section-kicker", appState.locale === "zh" ? "创建设备" : "Provision");
  setText("[data-view=\"devices\"] .overview-grid .panel:nth-of-type(1) .section-head h3", appState.locale === "zh" ? "新增设备" : "Create Device");
  setText("#device-form label:nth-of-type(1) span", appState.locale === "zh" ? "设备名称" : "Device Name");
  setText("#device-form label:nth-of-type(2) span", appState.locale === "zh" ? "所属产品" : "Product");
  setText("#device-form label:nth-of-type(3) span", appState.locale === "zh" ? "标签 JSON" : "Tags JSON");
  setText("#device-form label:nth-of-type(4) span", appState.locale === "zh" ? "元数据 JSON" : "Metadata JSON");
  setText("#device-form button[type=\"submit\"]", appState.locale === "zh" ? "创建设备" : "Create Device");
  setText("[data-view=\"devices\"] .overview-grid .panel:nth-of-type(2) .section-kicker", appState.locale === "zh" ? "设备" : "Fleet");
  setText("[data-view=\"devices\"] .overview-grid .panel:nth-of-type(2) .section-head h3", appState.locale === "zh" ? "设备列表" : "Device List");
  setText("[data-view=\"devices\"] > .panel .section-kicker", appState.locale === "zh" ? "详情" : "Detail");
  setText("[data-view=\"devices\"] > .panel .section-head h3", appState.locale === "zh" ? "当前设备" : "Selected Device");

  setText("[data-view=\"config\"] .section-kicker", appState.locale === "zh" ? "配置" : "Config");
  setText("[data-view=\"config\"] .section-head h3", appState.locale === "zh" ? "远程配置模板" : "Remote Config Profiles");
  setText("#config-form label:nth-of-type(1) span", appState.locale === "zh" ? "模板名称" : "Profile Name");
  setText("#config-form label:nth-of-type(2) span", appState.locale === "zh" ? "产品" : "Product");
  setText("#config-form label:nth-of-type(3) span", appState.locale === "zh" ? "描述" : "Description");
  setText("#config-form label:nth-of-type(4) span", appState.locale === "zh" ? "配置值 JSON" : "Values JSON");
  setText("#config-form button[type=\"submit\"]", appState.locale === "zh" ? "创建配置模板" : "Create Config Profile");

  setText("[data-view=\"simulator\"] .section-kicker", appState.locale === "zh" ? "模拟器" : "Simulator");
  setText("[data-view=\"simulator\"] .section-head h3", appState.locale === "zh" ? "设备模拟器" : "Device Simulator");
  setText("#sim-form label:nth-of-type(1) span", appState.locale === "zh" ? "模拟器设备名" : "Simulator Device Name");
  setText("#sim-form label:nth-of-type(2) span", appState.locale === "zh" ? "遥测间隔 ms" : "Telemetry Interval ms");
  setText("#sim-form label:nth-of-type(3) span", appState.locale === "zh" ? "产品" : "Product");
  setText("#sim-form label:nth-of-type(8) span", appState.locale === "zh" ? "默认遥测 JSON" : "Default Telemetry JSON");
  setText("#sim-form label:nth-of-type(9) span", appState.locale === "zh" ? "元数据 JSON" : "Metadata JSON");
  setText("#sim-form button[type=\"submit\"]", appState.locale === "zh" ? "创建模拟器" : "Create Simulator");

  setPlaceholder("product-name", appState.locale === "zh" ? "温湿度传感器产品" : "thermostat-product");
  setPlaceholder("product-description", appState.locale === "zh" ? "智能传感器产品" : "Smart sensor product");
  setPlaceholder("product-topic", appState.locale === "zh" ? "factory/+/sensor/+/up" : "factory/+/sensor/+/up");
  setPlaceholder("device-name", appState.locale === "zh" ? "边缘传感器-01" : "edge-sensor-01");
  setPlaceholder("group-name", appState.locale === "zh" ? "产线-A" : "line-a");
  setPlaceholder("group-description", appState.locale === "zh" ? "装配线 A" : "Assembly line A");
  setPlaceholder("rule-name", appState.locale === "zh" ? "温度过高" : "temperature-high");
  setPlaceholder("rule-property", appState.locale === "zh" ? "temperature" : "temperature");
  setPlaceholder("rule-description", appState.locale === "zh" ? "温度大于 30 时告警" : "Alert when temperature is higher than 30");
  setPlaceholder("config-name", appState.locale === "zh" ? "夜间模式" : "night-mode");
  setPlaceholder("config-description", appState.locale === "zh" ? "可复用的远程配置模板" : "Reusable desired-shadow template");
  setPlaceholder("sim-name", appState.locale === "zh" ? "虚拟电表-01" : "virtual-meter-01");
}

function getProduct(productId) {
  return appState.products.find((item) => item.product.id === productId) || null;
}

function getGroup(groupId) {
  return appState.groups.find((item) => item.group.id === groupId) || null;
}

function getDevice(deviceId) {
  return appState.devices.find((item) => item.device.id === deviceId) || null;
}

function buildThingModelTemplate(productView) {
  const template = {};
  for (const property of (productView?.product?.thing_model?.properties || [])) {
    switch (property.data_type) {
      case "bool":
        template[property.identifier] = false;
        break;
      case "int":
        template[property.identifier] = 0;
        break;
      case "float":
      case "double":
        template[property.identifier] = 0;
        break;
      case "object":
        template[property.identifier] = {};
        break;
      default:
        template[property.identifier] = "";
        break;
    }
  }
  return template;
}

function renderKVTags(values, emptyLabel = "No tags") {
  const entries = Object.entries(values || {});
  if (entries.length === 0) {
    return `<span class="subtle">${escapeHTML(emptyLabel)}</span>`;
  }
  return entries
    .map(([key, value]) => `<span class="tag">${escapeHTML(key)}=${escapeHTML(value)}</span>`)
    .join("");
}

function renderHealth(health) {
  const dot = document.getElementById("health-dot");
  const text = document.getElementById("health-text");
  if (!dot || !text) {
    return;
  }

  const healthy = health?.status === "ok";
  dot.classList.toggle("online", healthy);
  text.textContent = healthy ? t("runtime_healthy", { time: formatTime(health.time) }) : t("runtime_unavailable");
}

function renderStats(metrics) {
  const container = document.getElementById("stats-grid");
  if (!container) {
    return;
  }

  const cards = [
    [t("products_metric"), appState.products.length],
    [t("devices_metric"), metrics?.registered_devices || 0],
    [t("online_metric"), metrics?.online_devices || 0],
    [t("groups_metric"), appState.groups.length],
    [t("rules_metric"), appState.rules.length],
    [t("alerts_metric"), appState.alerts.length],
    [t("configs_metric"), appState.configProfiles.length],
    [t("connections_metric"), metrics?.total_connections || 0],
    [t("telemetry_metric"), metrics?.telemetry_received || 0],
    [t("command_ack_metric"), metrics?.command_acks || 0],
    [t("http_ingest_metric"), metrics?.ingress?.http_ingest_accepted || 0],
    [t("mqtt_metric"), metrics?.ingress?.mqtt_messages_received || 0],
    [t("samples_metric"), metrics?.storage?.telemetry_samples || 0],
    [t("persist_errors_metric"), metrics?.storage?.persist_errors || 0],
  ];

  container.innerHTML = cards.map(([name, value]) => `
    <article class="metric-card">
      <span class="metric-label">${name}</span>
      <strong class="metric-value">${value}</strong>
    </article>
  `).join("");
}

function renderOverview() {
  const deviceContainer = document.getElementById("overview-device-list");
  const alertContainer = document.getElementById("overview-alert-list");
  if (!deviceContainer || !alertContainer) {
    return;
  }

  const recentDevices = [...appState.devices]
    .sort((left, right) => new Date(right.device.created_at) - new Date(left.device.created_at))
    .slice(0, 5);
  const latestAlerts = [...appState.alerts]
    .sort((left, right) => new Date(right.triggered_at) - new Date(left.triggered_at))
    .slice(0, 5);

  if (recentDevices.length === 0) {
    deviceContainer.className = "stack empty";
    deviceContainer.textContent = t("no_devices");
  } else {
    deviceContainer.className = "stack";
    deviceContainer.innerHTML = recentDevices.map((item) => `
      <article class="detail-card">
        <div class="line">
          <div>
            <strong>${escapeHTML(item.device.name)}</strong>
            <div class="muted mono">${escapeHTML(item.device.id)}</div>
          </div>
          <span class="pill ${item.online ? "online" : "offline"}">${item.online ? t("online") : t("offline")}</span>
        </div>
        <div class="mini-list">
          ${item.product ? `<span class="tag">${escapeHTML(item.product.name)}</span>` : `<span class="tag">${escapeHTML(t("unbound"))}</span>`}
          ${(item.groups || []).map((group) => `<span class="tag">${escapeHTML(group.name)}</span>`).join("")}
        </div>
        <div class="list-actions">
          <button class="button ghost" type="button" data-overview-device="${item.device.id}">${escapeHTML(t("inspect_device"))}</button>
        </div>
      </article>
    `).join("");

    deviceContainer.querySelectorAll("[data-overview-device]").forEach((button) => {
      button.addEventListener("click", async () => {
        appState.selectedDeviceId = button.dataset.overviewDevice;
        activateView("devices");
        renderDevices();
        renderGroups();
        renderConfigProfiles();
        try {
          await refreshSelectedDevice();
        } catch (error) {
          handleGlobalError(error);
        }
      });
    });
  }

  if (latestAlerts.length === 0) {
    alertContainer.className = "stack empty";
    alertContainer.textContent = t("no_alerts");
  } else {
    alertContainer.className = "stack";
    alertContainer.innerHTML = latestAlerts.map((item) => `
      <article class="detail-card alert-card">
        <div class="line">
          <div>
            <strong>${escapeHTML(item.rule_name)}</strong>
            <div class="muted">${escapeHTML(item.device_name)}</div>
          </div>
          <span class="severity ${escapeHTML(item.severity)}">${escapeHTML(item.severity)}</span>
        </div>
        <div class="mini-list">
          <span class="tag">${escapeHTML(item.status || "new")}</span>
          <span class="tag">${escapeHTML(item.property)} ${escapeHTML(item.operator)} ${escapeHTML(formatValue(item.threshold))}</span>
          <span class="tag">Value ${escapeHTML(formatValue(item.value))}</span>
        </div>
        <div class="muted">${formatTime(item.triggered_at)}</div>
      </article>
    `).join("");
  }
}

function formatToggle(value) {
  if (appState.locale === "zh") {
    return value ? "启用" : "关闭";
  }
  return value ? "Enabled" : "Disabled";
}

function renderSystemSummary() {
  const container = document.getElementById("system-summary");
  if (!container) {
    return;
  }

  const info = appState.systemInfo;
  if (!info) {
    container.className = "detail-meta-grid empty";
    container.textContent = appState.locale === "zh" ? "正在加载系统信息..." : "Loading system info...";
    return;
  }

  const rows = [
    [appState.locale === "zh" ? "安装状态" : "Install", info.installed ? (appState.locale === "zh" ? "已安装" : "Installed") : (appState.locale === "zh" ? "未安装" : "Pending")],
    [appState.locale === "zh" ? "节点 ID" : "Node ID", info.node_id || "-"],
    [appState.locale === "zh" ? "节点角色" : "Node Role", info.role || "-"],
    [appState.locale === "zh" ? "备用模式" : "Standby", info.standby ? (appState.locale === "zh" ? "是" : "Yes") : (appState.locale === "zh" ? "否" : "No")],
    [appState.locale === "zh" ? "存储后端" : "Store Backend", info.store_backend || "-"],
    [appState.locale === "zh" ? "内嵌后台" : "Embedded UI", formatToggle(info.embedded_ui)],
    [appState.locale === "zh" ? "TCP 网关" : "TCP Gateway", formatToggle(info.gateway_enabled)],
    [appState.locale === "zh" ? "MQTT 服务" : "MQTT Broker", formatToggle(info.mqtt_enabled)],
    [appState.locale === "zh" ? "配置文件" : "Setup File", info.setup_path || "-"],
    [appState.locale === "zh" ? "数据文件" : "Store Path", info.store_persistence_path || "-"],
  ];

  container.className = "detail-meta-grid";
  container.innerHTML = rows.map(([name, value]) => `
    <article class="meta-tile">
      <span>${escapeHTML(name)}</span>
      <strong>${escapeHTML(formatValue(value))}</strong>
    </article>
  `).join("");
}

function syncSelect(id, options, emptyLabel) {
  const select = document.getElementById(id);
  if (!select) {
    return;
  }

  const currentValue = select.value;
  select.innerHTML = [`<option value="">${escapeHTML(emptyLabel)}</option>`]
    .concat(options.map((item) => `<option value="${item.value}">${escapeHTML(item.label)}</option>`))
    .join("");

  if (currentValue && options.some((item) => item.value === currentValue)) {
    select.value = currentValue;
  }
}

function syncFormOptions() {
  const productOptions = appState.products.map((item) => ({
    value: item.product.id,
    label: `${item.product.name} | ${item.product.key}`,
  }));
  syncSelect("device-product-id", productOptions, t("unbound"));
  syncSelect("sim-product-id", productOptions, t("unbound"));
  syncSelect("group-product-id", productOptions, t("any_product"));
  syncSelect("rule-product-id", productOptions, t("auto"));
  syncSelect("config-product-id", productOptions, t("optional"));
  syncSelect("rule-group-id", appState.groups.map((item) => ({ value: item.group.id, label: item.group.name })), t("optional"));
  syncSelect("rule-device-id", appState.devices.map((item) => ({ value: item.device.id, label: item.device.name })), t("optional"));
}

function renderProducts() {
  const container = document.getElementById("product-list");
  document.getElementById("product-count").textContent = `${appState.products.length}`;
  if (appState.products.length === 0) {
    container.className = "stack empty";
    container.textContent = t("no_products");
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.products.map((item) => `
    <article class="detail-card">
      <div class="line">
        <div>
          <strong>${escapeHTML(item.product.name)}</strong>
          <div class="muted mono">${escapeHTML(item.product.key)}</div>
          <div class="muted mono">${escapeHTML(item.product.id)}</div>
        </div>
        <span class="chip">${item.device_count} devices</span>
      </div>
      <div class="detail-meta-grid">
        <div class="meta-tile"><span>${escapeHTML(t("online_metric"))}</span><strong>${item.online_count}</strong></div>
        <div class="meta-tile"><span>Properties</span><strong>${(item.product.thing_model.properties || []).length}</strong></div>
        <div class="meta-tile"><span>Services</span><strong>${(item.product.thing_model.services || []).length}</strong></div>
        <div class="meta-tile"><span>Version</span><strong>${item.product.thing_model.version || 0}</strong></div>
      </div>
      <div class="mini-list">
        <span class="tag">${escapeHTML(t("transport"))} ${escapeHTML(item.product.access_profile?.transport || "tcp")}</span>
        <span class="tag">${escapeHTML(t("protocol"))} ${escapeHTML(item.product.access_profile?.protocol || "tcp_json")}</span>
        <span class="tag">${escapeHTML(t("ingest_mode"))} ${escapeHTML(item.product.access_profile?.ingest_mode || "gateway_tcp")}</span>
        <span class="tag">${escapeHTML(t("payload_format"))} ${escapeHTML(item.product.access_profile?.payload_format || "json_values")}</span>
      </div>
      <div class="tag-list">${renderKVTags(item.product.metadata, t("no_metadata"))}</div>
      <pre>${escapeHTML(pretty(item.product.thing_model))}</pre>
    </article>
  `).join("");
}

function renderProtocolCatalog() {
  const container = document.getElementById("protocol-catalog-list");
  if (!container) {
    return;
  }

  if (appState.protocolCatalog.length === 0) {
    container.className = "stack empty";
    container.textContent = t("protocol_templates_empty");
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.protocolCatalog.map((entry) => `
    <article class="detail-card">
      <div class="line">
        <div>
          <strong>${escapeHTML(entry.name)}</strong>
          <div class="muted">${escapeHTML(entry.description)}</div>
        </div>
        <button class="button ghost" type="button" data-apply-protocol-template="${entry.id}">${escapeHTML(t("apply_template"))}</button>
      </div>
      <div class="mini-list">
        <span class="tag">${escapeHTML(t("transport"))} ${escapeHTML(entry.transport)}</span>
        <span class="tag">${escapeHTML(t("protocol"))} ${escapeHTML(entry.protocol)}</span>
        <span class="tag">${escapeHTML(t("ingest_mode"))} ${escapeHTML(entry.ingest_mode)}</span>
        <span class="tag">${escapeHTML(t("payload_format"))} ${escapeHTML(entry.payload_format)}</span>
      </div>
      <div class="tag-list">
        ${(entry.common_sensors || []).map((sensor) => `<span class="tag">${escapeHTML(sensor)}</span>`).join("")}
      </div>
      <pre>${escapeHTML(pretty(entry.example_payload || {}))}</pre>
    </article>
  `).join("");

  container.querySelectorAll("[data-apply-protocol-template]").forEach((button) => {
    button.addEventListener("click", () => applyProtocolTemplate(button.dataset.applyProtocolTemplate));
  });
}

function renderDevices() {
  const container = document.getElementById("device-list");
  document.getElementById("device-count").textContent = `${appState.devices.length}`;
  if (appState.devices.length === 0) {
    container.className = "stack empty";
    container.textContent = t("no_devices");
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.devices.map((item) => `
    <article class="device-card ${item.device.id === appState.selectedDeviceId ? "active" : ""}">
      <button type="button" data-device-id="${item.device.id}">
        <div class="line">
          <strong>${escapeHTML(item.device.name)}</strong>
          <span class="pill ${item.online ? "online" : "offline"}">${item.online ? t("online") : t("offline")}</span>
        </div>
        <div class="muted mono">${escapeHTML(item.device.id)}</div>
        <div class="muted">${item.product ? `Product ${escapeHTML(item.product.name)}` : t("unbound")}</div>
        <div class="mini-list">${renderKVTags(item.device.tags, t("no_tags"))}</div>
        <div class="mini-list">${(item.groups || []).map((group) => `<span class="tag">${escapeHTML(group.name)}</span>`).join("")}</div>
        <div class="muted">Last seen ${formatTime(item.last_seen)}</div>
        <div class="muted mono">Token ${escapeHTML(item.device.token || "")}</div>
      </button>
    </article>
  `).join("");

  container.querySelectorAll("[data-device-id]").forEach((button) => {
    button.addEventListener("click", async () => {
      appState.selectedDeviceId = button.dataset.deviceId;
      renderDevices();
      renderGroups();
      renderConfigProfiles();
      try {
        await refreshSelectedDevice();
      } catch (error) {
        handleGlobalError(error);
      }
    });
  });
}

function renderGroups() {
  const container = document.getElementById("group-list");
  document.getElementById("group-count").textContent = `${appState.groups.length}`;
  if (appState.groups.length === 0) {
    container.className = "stack empty";
    container.textContent = t("no_groups");
    return;
  }

  const selectedDevice = getDevice(appState.selectedDeviceId);
  container.className = "stack";
  container.innerHTML = appState.groups.map((item) => {
    const member = selectedDevice && (selectedDevice.groups || []).some((group) => group.id === item.group.id);
    return `
      <article class="detail-card">
        <div class="line">
          <div>
            <strong>${escapeHTML(item.group.name)}</strong>
            <div class="muted mono">${escapeHTML(item.group.id)}</div>
            <div class="muted">${item.product ? `Product ${escapeHTML(item.product.name)}` : t("any_product")}</div>
          </div>
          <span class="chip">${item.device_count} devices</span>
        </div>
        <div class="detail-meta-grid">
          <div class="meta-tile"><span>Online</span><strong>${item.online_count}</strong></div>
          <div class="meta-tile"><span>Description</span><strong>${escapeHTML(item.group.description || "-")}</strong></div>
        </div>
        <div class="tag-list">${renderKVTags(item.group.tags, t("no_tags"))}</div>
        <div class="list-actions">
          ${selectedDevice
            ? `<button class="button ghost" type="button" data-group-${member ? "remove" : "add"}="${item.group.id}">${member ? escapeHTML(t("remove_selected_device")) : escapeHTML(t("add_selected_device"))}</button>`
            : `<span class="subtle">${escapeHTML(t("select_device_membership"))}</span>`}
        </div>
      </article>
    `;
  }).join("");

  container.querySelectorAll("[data-group-add]").forEach((button) => {
    button.addEventListener("click", () => updateGroupMembership(button.dataset.groupAdd, appState.selectedDeviceId, true));
  });
  container.querySelectorAll("[data-group-remove]").forEach((button) => {
    button.addEventListener("click", () => updateGroupMembership(button.dataset.groupRemove, appState.selectedDeviceId, false));
  });
}

function renderRules() {
  const container = document.getElementById("rule-list");
  document.getElementById("rule-count").textContent = `${appState.rules.length}`;
  if (appState.rules.length === 0) {
    container.className = "stack empty";
    container.textContent = t("no_rules");
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.rules.map((item) => `
    <article class="detail-card rule-card">
      <div class="line">
        <div>
          <strong>${escapeHTML(item.rule.name)}</strong>
          <div class="muted">${escapeHTML(item.rule.description || "Threshold rule")}</div>
        </div>
        <span class="severity ${escapeHTML(item.rule.severity)}">${escapeHTML(item.rule.severity)}</span>
      </div>
      <div class="detail-meta-grid">
        <div class="meta-tile"><span>Condition</span><strong class="mono">${escapeHTML(item.rule.condition.property)} ${escapeHTML(item.rule.condition.operator)} ${escapeHTML(formatValue(item.rule.condition.value))}</strong></div>
        <div class="meta-tile"><span>Triggered</span><strong>${item.triggered_count}</strong></div>
        <div class="meta-tile"><span>Last Triggered</span><strong>${formatTime(item.last_triggered_at)}</strong></div>
      </div>
      <div class="mini-list">
        ${item.product ? `<span class="tag">Product ${escapeHTML(item.product.name)}</span>` : ""}
        ${item.group ? `<span class="tag">Group ${escapeHTML(item.group.name)}</span>` : ""}
        ${item.device ? `<span class="tag">Device ${escapeHTML(item.device.name)}</span>` : ""}
        <span class="tag">Cooldown ${item.rule.cooldown_seconds || 0}s</span>
      </div>
    </article>
  `).join("");
}

function renderAlerts() {
  const container = document.getElementById("alert-list");
  document.getElementById("alert-count").textContent = `${appState.alerts.length}`;
  if (appState.alerts.length === 0) {
    container.className = "stack empty";
    container.textContent = t("no_alerts");
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.alerts.map((item) => `
    <article class="detail-card alert-card">
      <div class="line">
        <div>
          <strong>${escapeHTML(item.rule_name)}</strong>
          <div class="muted">${escapeHTML(item.message)}</div>
        </div>
        <span class="severity ${escapeHTML(item.severity)}">${escapeHTML(item.severity)}</span>
      </div>
      <div class="mini-list">
        <span class="tag">Status ${escapeHTML(item.status || "new")}</span>
        <span class="tag">Device ${escapeHTML(item.device_name)}</span>
        <span class="tag">${escapeHTML(item.property)} ${escapeHTML(item.operator)} ${escapeHTML(formatValue(item.threshold))}</span>
        <span class="tag">Value ${escapeHTML(formatValue(item.value))}</span>
      </div>
      <div class="detail-meta-grid">
        <div class="meta-tile"><span>Triggered</span><strong>${formatTime(item.triggered_at)}</strong></div>
        <div class="meta-tile"><span>Acknowledged</span><strong>${formatTime(item.ack_at)}</strong></div>
        <div class="meta-tile"><span>Resolved</span><strong>${formatTime(item.resolved_at)}</strong></div>
      </div>
      <div class="muted">${escapeHTML(item.note || "No note")}</div>
      <div class="list-actions">
        ${item.status !== "acknowledged" && item.status !== "resolved" ? `<button class="button ghost" type="button" data-alert-ack="${item.id}">${escapeHTML(t("acknowledged"))}</button>` : ""}
        ${item.status !== "resolved" ? `<button class="button primary" type="button" data-alert-resolve="${item.id}">${escapeHTML(t("resolved"))}</button>` : ""}
      </div>
    </article>
  `).join("");

  container.querySelectorAll("[data-alert-ack]").forEach((button) => {
    button.addEventListener("click", () => updateAlertStatus(button.dataset.alertAck, "acknowledged"));
  });
  container.querySelectorAll("[data-alert-resolve]").forEach((button) => {
    button.addEventListener("click", () => updateAlertStatus(button.dataset.alertResolve, "resolved"));
  });
}

function renderConfigProfiles() {
  const container = document.getElementById("config-list");
  document.getElementById("config-count").textContent = `${appState.configProfiles.length}`;
  if (appState.configProfiles.length === 0) {
    container.className = "stack empty";
    container.textContent = t("no_configs");
    return;
  }

  const selectedDevice = getDevice(appState.selectedDeviceId);
  const selectedProductID = selectedDevice?.device?.product_id || "";
  container.className = "stack";
  container.innerHTML = appState.configProfiles.map((item) => {
    const scopedToAnotherProduct = !!item.product && item.product.id !== selectedProductID;
    const canApply = !!selectedDevice && !scopedToAnotherProduct;
    return `
      <article class="detail-card">
        <div class="line">
          <div>
            <strong>${escapeHTML(item.profile.name)}</strong>
            <div class="muted mono">${escapeHTML(item.profile.id)}</div>
            <div class="muted">${escapeHTML(item.profile.description || "Reusable desired-shadow profile")}</div>
          </div>
          <span class="chip">${item.profile.applied_count || 0} applied</span>
        </div>
        <div class="mini-list">
          ${item.product ? `<span class="tag">Product ${escapeHTML(item.product.name)}</span>` : `<span class="tag">${escapeHTML(t("all_products"))}</span>`}
          <span class="tag">${escapeHTML(t("updated"))} ${escapeHTML(formatTime(item.profile.updated_at))}</span>
        </div>
        <div class="list-actions">
          ${canApply ? `<button class="button primary" type="button" data-config-apply="${item.profile.id}">${escapeHTML(t("apply_selected_device"))}</button>` : ""}
          ${!selectedDevice ? `<span class="subtle">${escapeHTML(t("select_device_apply"))}</span>` : ""}
          ${scopedToAnotherProduct ? `<span class="subtle">${escapeHTML(t("product_scope_mismatch"))}</span>` : ""}
        </div>
        <pre>${escapeHTML(pretty(item.profile.values || {}))}</pre>
      </article>
    `;
  }).join("");

  container.querySelectorAll("[data-config-apply]").forEach((button) => {
    button.addEventListener("click", () => applyConfigProfile(button.dataset.configApply, appState.selectedDeviceId));
  });
}

async function refreshSelectedDevice() {
  const panel = document.getElementById("device-detail");
  const selectedName = document.getElementById("selected-device-name");
  if (!panel || !selectedName) {
    return;
  }

  if (!appState.selectedDeviceId) {
    selectedName.textContent = t("selected_device_none");
    panel.className = "stack empty";
    panel.textContent = t("selected_device_empty");
    return;
  }

  const [device, shadow, telemetry, commands, alerts] = await Promise.all([
    requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}`),
    requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/shadow`),
    requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/telemetry?limit=20`),
    requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/commands?limit=20`),
    requestJSON(`/api/v1/alerts?device_id=${encodeURIComponent(appState.selectedDeviceId)}&limit=10`),
  ]);
  const deviceProductView = getProduct(device.device.product_id);
  const accessProfile = deviceProductView?.product?.access_profile || {};

  selectedName.textContent = device.device.name;
  panel.className = "device-detail-grid";
  panel.innerHTML = `
    <article class="detail-card">
      <div class="line">
        <div>
          <strong>${escapeHTML(device.device.name)}</strong>
          <div class="muted mono">${escapeHTML(device.device.id)}</div>
        </div>
        <span class="pill ${device.online ? "online" : "offline"}">${device.online ? t("online") : t("offline")}</span>
      </div>
      <div class="mini-list">
        ${device.product ? `<span class="tag">Product ${escapeHTML(device.product.name)}</span>` : `<span class="tag">${escapeHTML(t("unbound"))}</span>`}
        ${(device.groups || []).map((group) => `<span class="tag">${escapeHTML(group.name)}</span>`).join("")}
        <span class="tag">${escapeHTML(t("protocol"))} ${escapeHTML(accessProfile.protocol || "tcp_json")}</span>
        <span class="tag">${escapeHTML(t("ingest_mode"))} ${escapeHTML(accessProfile.ingest_mode || "gateway_tcp")}</span>
      </div>
      <div class="detail-meta-grid">
        <div class="meta-tile"><span>Connected</span><strong>${formatTime(device.connected_at)}</strong></div>
        <div class="meta-tile"><span>Last Seen</span><strong>${formatTime(device.last_seen)}</strong></div>
        <div class="meta-tile"><span>Token</span><strong class="mono">${escapeHTML(device.device.token || "-")}</strong></div>
      </div>
      <div class="tag-list">${renderKVTags(device.device.tags, t("no_tags"))}</div>
      <pre>${escapeHTML(pretty({
        transport: accessProfile.transport || "tcp",
        protocol: accessProfile.protocol || "tcp_json",
        ingest_mode: accessProfile.ingest_mode || "gateway_tcp",
        payload_format: accessProfile.payload_format || "json_values",
        http_push_path: buildHTTPIngestPath(device.device.id),
      }))}</pre>
      <pre>${escapeHTML(pretty(device.device.metadata || {}))}</pre>
    </article>

    <article class="detail-card">
      <div class="line"><strong>${escapeHTML(appState.locale === "zh" ? "设备标签" : "Device Tags")}</strong></div>
      <form id="device-tags-form" class="grid-form">
        <label class="wide">
          <span>${escapeHTML(appState.locale === "zh" ? "标签 JSON" : "Tags JSON")}</span>
          <textarea id="device-tags-editor" rows="5">${escapeHTML(pretty(device.device.tags || {}))}</textarea>
        </label>
        <div class="actions wide">
          <button class="button ghost" type="submit">${escapeHTML(appState.locale === "zh" ? "更新标签" : "Update Tags")}</button>
          <span id="device-tags-status" class="hint"></span>
        </div>
      </form>
    </article>

    <article class="detail-card">
      <div class="line"><strong>${escapeHTML(appState.locale === "zh" ? "远程配置" : "Remote Config")}</strong></div>
      <form id="device-config-form" class="grid-form">
        <label class="wide">
          <span>${escapeHTML(appState.locale === "zh" ? "配置模板" : "Config Profile")}</span>
          <select id="device-config-profile-id">
            <option value="">${escapeHTML(appState.locale === "zh" ? "选择模板" : "Select profile")}</option>
            ${appState.configProfiles.map((item) => `<option value="${item.profile.id}">${escapeHTML(item.profile.name)}${item.product ? ` | ${escapeHTML(item.product.name)}` : ""}</option>`).join("")}
          </select>
        </label>
        <div class="actions wide">
          <button class="button primary" type="submit">${escapeHTML(appState.locale === "zh" ? "下发配置模板" : "Apply Config Profile")}</button>
          <span id="device-config-status" class="hint"></span>
        </div>
      </form>
      <pre>${escapeHTML(pretty(shadow.desired || {}))}</pre>
    </article>

    <article class="detail-card">
      <div class="line"><strong>${escapeHTML(appState.locale === "zh" ? "设备影子" : "Device Shadow")}</strong></div>
      <form id="shadow-form" class="grid-form">
        <label class="wide">
          <span>${escapeHTML(appState.locale === "zh" ? "期望值 JSON" : "Desired JSON")}</span>
          <textarea id="shadow-desired" rows="6">${escapeHTML(pretty(shadow.desired || {}))}</textarea>
        </label>
        <div class="actions wide">
          <button class="button ghost" type="submit">${escapeHTML(appState.locale === "zh" ? "更新期望值" : "Update Desired")}</button>
          <span id="shadow-status" class="hint"></span>
        </div>
      </form>
      <pre>${escapeHTML(pretty({ reported: shadow.reported || {}, desired: shadow.desired || {}, updated_at: shadow.updated_at }))}</pre>
    </article>

    <article class="detail-card">
      <div class="line"><strong>${escapeHTML(appState.locale === "zh" ? "下发命令" : "Send Command")}</strong></div>
      <form id="command-form" class="grid-form">
        <label>
          <span>${escapeHTML(appState.locale === "zh" ? "命令" : "Command")}</span>
          <input id="command-name" type="text" value="reboot">
        </label>
        <label class="wide">
          <span>${escapeHTML(appState.locale === "zh" ? "参数 JSON" : "Params JSON")}</span>
          <textarea id="command-params" rows="4">{"delay":1}</textarea>
        </label>
        <div class="actions wide">
          <button class="button primary" type="submit">${escapeHTML(appState.locale === "zh" ? "发送命令" : "Send Command")}</button>
          <span id="command-status" class="hint"></span>
        </div>
      </form>
    </article>

    <article class="detail-card">
      <div class="line"><strong>${escapeHTML(appState.locale === "zh" ? "最近遥测" : "Recent Telemetry")}</strong></div>
      <pre>${escapeHTML(pretty(telemetry))}</pre>
    </article>

    <article class="detail-card">
      <div class="line"><strong>${escapeHTML(appState.locale === "zh" ? "最近命令" : "Recent Commands")}</strong></div>
      <pre>${escapeHTML(pretty(commands))}</pre>
    </article>

    <article class="detail-card">
      <div class="line"><strong>${escapeHTML(appState.locale === "zh" ? "设备告警" : "Device Alerts")}</strong></div>
      <pre>${escapeHTML(pretty(alerts))}</pre>
    </article>
  `;

  document.getElementById("device-tags-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("device-tags-status", t("update_in_progress"));
      await requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/tags`, {
        method: "PUT",
        body: JSON.stringify({ tags: parseJSON(document.getElementById("device-tags-editor").value, {}) }),
      });
      setHint("device-tags-status", t("tags_updated"));
      await refreshAll();
    } catch (error) {
      setHint("device-tags-status", error.message, true);
    }
  });

  document.getElementById("device-config-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const profileID = document.getElementById("device-config-profile-id").value;
      if (!profileID) {
        throw new Error(t("select_profile_first"));
      }
      setHint("device-config-status", t("apply_in_progress"));
      await applyConfigProfile(profileID, appState.selectedDeviceId, "device-config-status");
    } catch (error) {
      setHint("device-config-status", error.message, true);
    }
  });

  document.getElementById("shadow-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("shadow-status", t("update_in_progress"));
      await requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/shadow`, {
        method: "PUT",
        body: JSON.stringify({ desired: parseJSON(document.getElementById("shadow-desired").value, {}) }),
      });
      setHint("shadow-status", t("shadow_updated"));
      await refreshAll();
    } catch (error) {
      setHint("shadow-status", error.message, true);
    }
  });

  document.getElementById("command-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("command-status", t("send_in_progress"));
      await requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/commands`, {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("command-name").value.trim(),
          params: parseJSON(document.getElementById("command-params").value, {}),
        }),
      });
      setHint("command-status", t("command_accepted"));
      await refreshAll();
    } catch (error) {
      setHint("command-status", error.message, true);
    }
  });
}

function renderSimulators() {
  const container = document.getElementById("sim-list");
  if (!container) {
    return;
  }

  if (appState.simulators.length === 0) {
    container.className = "stack empty";
    container.textContent = t("no_simulators");
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.simulators.map((sim) => `
    <article class="sim-card">
      <div class="line">
        <div>
          <strong>${escapeHTML(sim.device.name)}</strong>
          <div class="muted mono">${escapeHTML(sim.device.id)}</div>
          <div class="muted">${sim.device.product_key ? `ProductKey ${escapeHTML(sim.device.product_key)}` : "Unbound"}</div>
        </div>
        <div class="sim-summary">
          <span class="pill ${sim.connected ? "online" : "offline"}">${sim.connected ? t("online") : t("offline")}</span>
          <span class="pill">ack ${sim.auto_ack ? "on" : "off"}</span>
          <span class="pill">ping ${sim.auto_ping ? "on" : "off"}</span>
          <span class="pill">telemetry ${sim.auto_telemetry ? `${sim.telemetry_interval_ms}ms` : "manual"}</span>
        </div>
      </div>
      <div class="sim-actions">
        <button class="button ghost" type="button" data-connect="${sim.id}">${escapeHTML(t("connect"))}</button>
        <button class="button ghost" type="button" data-disconnect="${sim.id}">${escapeHTML(t("disconnect"))}</button>
        <button class="button accent" type="button" data-telemetry="${sim.id}">${escapeHTML(t("send_telemetry"))}</button>
        <button class="button ghost" type="button" data-remove="${sim.id}">${escapeHTML(t("remove"))}</button>
      </div>
      <div class="sim-grid">
        <div class="detail-card">
          <div class="line"><strong>Config</strong></div>
          <pre>${escapeHTML(pretty({
            auto_ack: sim.auto_ack,
            auto_ping: sim.auto_ping,
            auto_telemetry: sim.auto_telemetry,
            telemetry_interval_ms: sim.telemetry_interval_ms,
            default_values: sim.default_values || {},
          }))}</pre>
        </div>
        <div class="detail-card">
          <div class="line"><strong>Status</strong></div>
          <pre>${escapeHTML(pretty({
            last_connect_at: sim.last_connect_at,
            last_disconnect_at: sim.last_disconnect_at,
            last_ping_at: sim.last_ping_at,
            last_telemetry_at: sim.last_telemetry_at,
            last_command_at: sim.last_command_at,
            last_error: sim.last_error || "",
          }))}</pre>
        </div>
        <div class="detail-card">
          <div class="line"><strong>Logs</strong></div>
          <pre>${escapeHTML((sim.logs || []).map((entry) => `[${formatTime(entry.timestamp)}] ${entry.level.toUpperCase()} ${entry.message}`).join("\n") || t("no_logs"))}</pre>
        </div>
      </div>
    </article>
  `).join("");

  document.querySelectorAll("[data-connect]").forEach((button) => {
    button.addEventListener("click", () => doSimulatorAction(button.dataset.connect, "connect"));
  });
  document.querySelectorAll("[data-disconnect]").forEach((button) => {
    button.addEventListener("click", () => doSimulatorAction(button.dataset.disconnect, "disconnect"));
  });
  document.querySelectorAll("[data-remove]").forEach((button) => {
    button.addEventListener("click", () => doSimulatorAction(button.dataset.remove, "remove"));
  });
  document.querySelectorAll("[data-telemetry]").forEach((button) => {
    button.addEventListener("click", () => doSimulatorAction(button.dataset.telemetry, "telemetry"));
  });
}

async function doSimulatorAction(id, action) {
  try {
    if (action === "connect") {
      await requestJSON(`/api/v1/simulators/${encodeURIComponent(id)}/connect`, { method: "POST", body: "{}" });
    } else if (action === "disconnect") {
      await requestJSON(`/api/v1/simulators/${encodeURIComponent(id)}/disconnect`, { method: "POST", body: "{}" });
    } else if (action === "remove") {
      await requestJSON(`/api/v1/simulators/${encodeURIComponent(id)}`, { method: "DELETE" });
    } else {
      const sim = appState.simulators.find((item) => item.id === id);
      await requestJSON(`/api/v1/simulators/${encodeURIComponent(id)}/telemetry`, {
        method: "POST",
        body: JSON.stringify({ values: sim ? sim.default_values : {} }),
      });
    }
    await refreshAll();
  } catch (error) {
    window.alert(error.message);
  }
}

async function updateGroupMembership(groupId, deviceId, add) {
  if (!groupId || !deviceId) {
    return;
  }

  try {
    if (add) {
      await requestJSON(`/api/v1/groups/${encodeURIComponent(groupId)}/devices`, {
        method: "POST",
        body: JSON.stringify({ device_id: deviceId }),
      });
    } else {
      await requestJSON(`/api/v1/groups/${encodeURIComponent(groupId)}/devices/${encodeURIComponent(deviceId)}`, {
        method: "DELETE",
      });
    }
    await refreshAll();
  } catch (error) {
    window.alert(error.message);
  }
}

async function applyConfigProfile(profileId, deviceId, hintId = "config-status") {
  if (!profileId || !deviceId) {
    throw new Error("config profile and device are required");
  }

  setHint(hintId, t("apply_in_progress"));
  await requestJSON(`/api/v1/config-profiles/${encodeURIComponent(profileId)}/apply`, {
    method: "POST",
    body: JSON.stringify({ device_id: deviceId }),
  });
  setHint(hintId, t("config_applied"));
  await refreshAll();
}

function applyProtocolTemplate(templateId) {
  const entry = appState.protocolCatalog.find((item) => item.id === templateId);
  if (!entry) {
    return;
  }

  document.getElementById("product-transport").value = entry.access_profile.transport || "tcp";
  document.getElementById("product-protocol").value = entry.access_profile.protocol || "tcp_json";
  document.getElementById("product-ingest-mode").value = entry.access_profile.ingest_mode || "gateway_tcp";
  document.getElementById("product-payload-format").value = entry.access_profile.payload_format || "json_values";
  document.getElementById("product-sensor-template").value = entry.access_profile.sensor_template || "generic";
  document.getElementById("product-auth-mode").value = entry.access_profile.auth_mode || "token";
  document.getElementById("product-topic").value = entry.access_profile.topic || "";
  document.getElementById("product-point-mappings").value = pretty(entry.access_profile.point_mappings || []);
  document.getElementById("product-thing-model").value = pretty(entry.thing_model || {});
  document.getElementById("product-metadata").value = pretty({
    template_id: entry.id,
    template_name: entry.name,
  });
  setHint("product-status", `${t("protocol_template")}: ${entry.name}`);
}

async function updateAlertStatus(alertId, status) {
  const note = window.prompt("Processing note", "");
  if (note === null) {
    return;
  }

  try {
    await requestJSON(`/api/v1/alerts/${encodeURIComponent(alertId)}`, {
      method: "PUT",
      body: JSON.stringify({ status, note }),
    });
    await refreshAll();
  } catch (error) {
    window.alert(error.message);
  }
}

function activateView(viewId) {
  const target = VIEW_TITLE_KEYS[viewId] ? viewId : "overview";
  appState.currentView = target;

  document.querySelectorAll("[data-view-target]").forEach((node) => {
    node.classList.toggle("active", node.dataset.viewTarget === target);
  });
  document.querySelectorAll("[data-view]").forEach((node) => {
    node.classList.toggle("active", node.dataset.view === target);
  });

  const title = document.getElementById("view-title");
  if (title) {
    title.textContent = t(VIEW_TITLE_KEYS[target]);
  }
}

async function refreshAll() {
  const keepCurrentDetail = isEditingTextField();
  const [health, metrics, systemInfo, catalog, products, devices, groups, rules, alerts, configProfiles, simulators] = await Promise.all([
    requestJSON("/healthz"),
    requestJSON("/metrics"),
    requestJSON("/api/v1/system/info"),
    requestJSON("/api/v1/protocol-catalog"),
    requestJSON("/api/v1/products"),
    requestJSON("/api/v1/devices"),
    requestJSON("/api/v1/groups"),
    requestJSON("/api/v1/rules"),
    requestJSON("/api/v1/alerts?limit=20"),
    requestJSON("/api/v1/config-profiles"),
    requestJSON("/api/v1/simulators"),
  ]);

  appState.health = health;
  appState.metrics = metrics;
  appState.systemInfo = systemInfo;
  appState.protocolCatalog = catalog;
  appState.products = products;
  appState.devices = devices;
  appState.groups = groups;
  appState.rules = rules;
  appState.alerts = alerts;
  appState.configProfiles = configProfiles;
  appState.simulators = simulators;

  if (!appState.selectedDeviceId && devices.length > 0) {
    appState.selectedDeviceId = devices[0].device.id;
  }
  if (appState.selectedDeviceId && !devices.some((item) => item.device.id === appState.selectedDeviceId)) {
    appState.selectedDeviceId = devices.length > 0 ? devices[0].device.id : "";
  }

  renderHealth(health);
  renderStats(metrics);
  syncFormOptions();
  renderOverview();
  renderSystemSummary();
  renderProducts();
  renderProtocolCatalog();
  renderDevices();
  renderGroups();
  renderRules();
  renderAlerts();
  renderConfigProfiles();
  renderSimulators();
  applyTranslations();

  if (!keepCurrentDetail) {
    await refreshSelectedDevice();
  }
}

function bindNavigation() {
  document.querySelectorAll("[data-view-target]").forEach((button) => {
    button.addEventListener("click", async () => {
      activateView(button.dataset.viewTarget);
      if (button.dataset.viewTarget === "devices") {
        try {
          await refreshSelectedDevice();
        } catch (error) {
          handleGlobalError(error);
        }
      }
    });
  });
}

function bindForms() {
  document.getElementById("product-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("product-status", t("create_in_progress"));
      await requestJSON("/api/v1/products", {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("product-name").value.trim(),
          description: document.getElementById("product-description").value.trim(),
          metadata: parseJSON(document.getElementById("product-metadata").value, {}),
          access_profile: {
            transport: document.getElementById("product-transport").value,
            protocol: document.getElementById("product-protocol").value,
            ingest_mode: document.getElementById("product-ingest-mode").value,
            payload_format: document.getElementById("product-payload-format").value,
            sensor_template: document.getElementById("product-sensor-template").value,
            auth_mode: document.getElementById("product-auth-mode").value,
            topic: document.getElementById("product-topic").value.trim(),
            point_mappings: parseJSON(document.getElementById("product-point-mappings").value, []),
          },
          thing_model: parseJSON(document.getElementById("product-thing-model").value, {}),
        }),
      });
      document.getElementById("product-name").value = "";
      document.getElementById("product-description").value = "";
      setHint("product-status", t("created_ok"));
      await refreshAll();
    } catch (error) {
      setHint("product-status", error.message, true);
    }
  });

  document.getElementById("device-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("device-status", t("create_in_progress"));
      await requestJSON("/api/v1/devices", {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("device-name").value.trim(),
          product_id: document.getElementById("device-product-id").value,
          tags: parseJSON(document.getElementById("device-tags").value, {}),
          metadata: parseJSON(document.getElementById("device-metadata").value, {}),
        }),
      });
      document.getElementById("device-name").value = "";
      setHint("device-status", t("created_ok"));
      await refreshAll();
    } catch (error) {
      setHint("device-status", error.message, true);
    }
  });

  document.getElementById("group-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("group-status", t("create_in_progress"));
      await requestJSON("/api/v1/groups", {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("group-name").value.trim(),
          description: document.getElementById("group-description").value.trim(),
          product_id: document.getElementById("group-product-id").value,
          tags: parseJSON(document.getElementById("group-tags").value, {}),
        }),
      });
      document.getElementById("group-name").value = "";
      document.getElementById("group-description").value = "";
      setHint("group-status", t("created_ok"));
      await refreshAll();
    } catch (error) {
      setHint("group-status", error.message, true);
    }
  });

  document.getElementById("rule-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("rule-status", t("create_in_progress"));
      await requestJSON("/api/v1/rules", {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("rule-name").value.trim(),
          description: document.getElementById("rule-description").value.trim(),
          product_id: document.getElementById("rule-product-id").value,
          group_id: document.getElementById("rule-group-id").value,
          device_id: document.getElementById("rule-device-id").value,
          severity: document.getElementById("rule-severity").value,
          cooldown_seconds: Number(document.getElementById("rule-cooldown").value || 0),
          condition: {
            property: document.getElementById("rule-property").value.trim(),
            operator: document.getElementById("rule-operator").value,
            value: parseLooseValue(document.getElementById("rule-threshold").value),
          },
        }),
      });
      document.getElementById("rule-name").value = "";
      document.getElementById("rule-description").value = "";
      setHint("rule-status", t("created_ok"));
      await refreshAll();
    } catch (error) {
      setHint("rule-status", error.message, true);
    }
  });

  document.getElementById("config-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("config-status", t("create_in_progress"));
      await requestJSON("/api/v1/config-profiles", {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("config-name").value.trim(),
          description: document.getElementById("config-description").value.trim(),
          product_id: document.getElementById("config-product-id").value,
          values: parseJSON(document.getElementById("config-values").value, {}),
        }),
      });
      document.getElementById("config-name").value = "";
      document.getElementById("config-description").value = "";
      setHint("config-status", t("created_ok"));
      await refreshAll();
    } catch (error) {
      setHint("config-status", error.message, true);
    }
  });

  document.getElementById("sim-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("sim-status", t("create_in_progress"));
      await requestJSON("/api/v1/simulators", {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("sim-name").value.trim(),
          product_id: document.getElementById("sim-product-id").value,
          telemetry_interval_ms: Number(document.getElementById("sim-interval").value || 5000),
          auto_connect: document.getElementById("sim-auto-connect").checked,
          auto_ack: document.getElementById("sim-auto-ack").checked,
          auto_ping: document.getElementById("sim-auto-ping").checked,
          auto_telemetry: document.getElementById("sim-auto-telemetry").checked,
          default_values: parseJSON(document.getElementById("sim-values").value, {}),
          metadata: parseJSON(document.getElementById("sim-metadata").value, {}),
        }),
      });
      document.getElementById("sim-name").value = "";
      setHint("sim-status", t("created_ok"));
      await refreshAll();
    } catch (error) {
      setHint("sim-status", error.message, true);
    }
  });

  document.getElementById("refresh-button").addEventListener("click", () => {
    refreshAll().catch(handleGlobalError);
  });
  document.getElementById("locale-toggle").addEventListener("click", () => {
    appState.locale = appState.locale === "zh" ? "en" : "zh";
    window.localStorage.setItem("mvp_locale", appState.locale);
    syncFormOptions();
    applyTranslations();
    renderStats(appState.metrics || {});
    renderOverview();
    renderProducts();
    renderProtocolCatalog();
    renderDevices();
    renderGroups();
    renderRules();
    renderAlerts();
    renderConfigProfiles();
    renderSimulators();
    refreshSelectedDevice().catch(handleGlobalError);
  });

  document.getElementById("sim-product-id").addEventListener("change", (event) => {
    const product = getProduct(event.target.value);
    if (product) {
      document.getElementById("sim-values").value = pretty(buildThingModelTemplate(product));
    }
  });

  document.getElementById("config-product-id").addEventListener("change", (event) => {
    const product = getProduct(event.target.value);
    if (product && !document.getElementById("config-values").value.trim()) {
      document.getElementById("config-values").value = pretty(buildThingModelTemplate(product));
    }
  });

  document.getElementById("rule-group-id").addEventListener("change", (event) => {
    const group = getGroup(event.target.value);
    if (group?.product) {
      document.getElementById("rule-product-id").value = group.product.id;
    }
  });

  document.getElementById("rule-device-id").addEventListener("change", (event) => {
    const device = getDevice(event.target.value);
    if (device?.product) {
      document.getElementById("rule-product-id").value = device.product.id;
    }
  });

  document.getElementById("rule-product-id").addEventListener("change", (event) => {
    const product = getProduct(event.target.value);
    const firstProperty = product?.product?.thing_model?.properties?.[0];
    if (firstProperty && !document.getElementById("rule-property").value.trim()) {
      document.getElementById("rule-property").value = firstProperty.identifier;
    }
  });
}

function handleGlobalError(error) {
  console.error(error);
  const healthText = document.getElementById("health-text");
  if (healthText) {
    healthText.textContent = error.message;
  }
}

async function bootstrap() {
  const installed = await ensureInstalled();
  if (!installed) {
    return;
  }
  bindNavigation();
  bindForms();
  activateView(appState.currentView);
  await refreshAll();
  window.setInterval(() => {
    refreshAll().catch(handleGlobalError);
  }, 4000);
}

bootstrap().catch(handleGlobalError);
