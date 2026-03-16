Object.assign(I18N.zh, {
  tenants_metric: "租户",
  firmware_metric: "固件",
  ota_metric: "OTA 任务",
  all_tenants: "全部租户",
  no_tenants: "暂无租户。",
  tenant_workspace: "租户工作区",
  tenant_name: "租户名称",
  tenant_slug: "租户标识",
  tenant_filter: "租户筛选",
  selected_tenant: "当前租户",
  rule_actions: "规则动作",
  action_type: "动作类型",
  command_name: "命令名称",
  command_params: "命令参数 JSON",
  alert_message: "告警消息",
  firmware_repo: "固件仓库",
  firmware_name: "固件名称",
  firmware_version: "版本",
  firmware_url: "下载地址",
  firmware_count: "固件",
  ota_count: "OTA",
  no_firmware: "暂无固件。",
  no_ota: "暂无 OTA 任务。",
  create_tenant: "创建租户",
  create_firmware: "创建固件",
  create_ota: "创建 OTA 任务",
  select_firmware: "选择固件",
  target_scope: "目标范围",
  total_devices: "设备总数",
  dispatched: "已下发",
  acked: "已回执",
  failed: "失败",
});

Object.assign(I18N.en, {
  tenants_metric: "Tenants",
  firmware_metric: "Firmware",
  ota_metric: "OTA Campaigns",
  all_tenants: "All Tenants",
  no_tenants: "No tenants yet.",
  tenant_workspace: "Tenant Workspace",
  tenant_name: "Tenant Name",
  tenant_slug: "Tenant Slug",
  tenant_filter: "Tenant Filter",
  selected_tenant: "Selected Tenant",
  rule_actions: "Rule Actions",
  action_type: "Action Type",
  command_name: "Command Name",
  command_params: "Command Params JSON",
  alert_message: "Alert Message",
  firmware_repo: "Firmware Repository",
  firmware_name: "Firmware Name",
  firmware_version: "Version",
  firmware_url: "Download URL",
  firmware_count: "Firmware",
  ota_count: "OTA",
  no_firmware: "No firmware artifacts yet.",
  no_ota: "No OTA campaigns yet.",
  create_tenant: "Create Tenant",
  create_firmware: "Create Firmware",
  create_ota: "Create OTA Campaign",
  select_firmware: "Select firmware",
  target_scope: "Target Scope",
  total_devices: "Total Devices",
  dispatched: "Dispatched",
  acked: "Acked",
  failed: "Failed",
});

appState.tenants = appState.tenants || [];
appState.firmwareArtifacts = appState.firmwareArtifacts || [];
appState.otaCampaigns = appState.otaCampaigns || [];
appState.selectedTenantId = window.localStorage.getItem("mvp_selected_tenant") || "";

function selectedTenantId() {
  return String(appState.selectedTenantId || "").trim();
}

function saveTenantSelection(value) {
  appState.selectedTenantId = String(value || "").trim();
  window.localStorage.setItem("mvp_selected_tenant", appState.selectedTenantId);
}

function withTenantQuery(path, params = {}) {
  const search = new URLSearchParams();
  const tenantID = selectedTenantId();
  if (tenantID) {
    search.set("tenant_id", tenantID);
  }
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") {
      return;
    }
    search.set(key, String(value));
  });
  const query = search.toString();
  return query ? `${path}?${query}` : path;
}

function selectedTenantName() {
  const tenant = appState.tenants.find((item) => item.tenant.id === selectedTenantId());
  return tenant ? tenant.tenant.name : t("all_tenants");
}

function syncTenantSelector() {
  const select = document.getElementById("tenant-selector");
  if (!select) {
    return;
  }

  const currentValue = selectedTenantId();
  select.innerHTML = [`<option value="">${escapeHTML(t("all_tenants"))}</option>`]
    .concat(appState.tenants.map((item) => `<option value="${item.tenant.id}">${escapeHTML(item.tenant.name)} | ${escapeHTML(item.tenant.slug)}</option>`))
    .join("");
  select.value = currentValue;
}

function syncEnhancedSelects() {
  syncTenantSelector();
  syncSelect(
    "rule-action-config-id",
    appState.configProfiles.map((item) => ({
      value: item.profile.id,
      label: item.profile.name,
    })),
    t("optional"),
  );
  syncSelect(
    "firmware-product-id",
    appState.products.map((item) => ({
      value: item.product.id,
      label: `${item.product.name} | ${item.product.key}`,
    })),
    t("optional"),
  );
  syncSelect(
    "ota-firmware-id",
    appState.firmwareArtifacts.map((item) => ({
      value: item.artifact.id,
      label: `${item.artifact.name} | ${item.artifact.version}`,
    })),
    t("select_firmware"),
  );
  syncSelect(
    "ota-product-id",
    appState.products.map((item) => ({
      value: item.product.id,
      label: `${item.product.name} | ${item.product.key}`,
    })),
    t("optional"),
  );
  syncSelect(
    "ota-group-id",
    appState.groups.map((item) => ({
      value: item.group.id,
      label: item.group.name,
    })),
    t("optional"),
  );
  syncSelect(
    "ota-device-id",
    appState.devices.map((item) => ({
      value: item.device.id,
      label: item.device.name,
    })),
    t("optional"),
  );
  syncRuleActionVisibility();
}

function syncRuleActionVisibility() {
  const type = document.getElementById("rule-action-type")?.value || "alert";
  const showCommand = type === "send_command";
  const showConfig = type === "apply_config_profile";
  const showMessage = type === "alert";

  document.getElementById("rule-action-command-wrap")?.classList.toggle("field-hidden", !showCommand);
  document.getElementById("rule-action-params-wrap")?.classList.toggle("field-hidden", !showCommand);
  document.getElementById("rule-action-config-wrap")?.classList.toggle("field-hidden", !showConfig);
  document.getElementById("rule-action-message-wrap")?.classList.toggle("field-hidden", !showMessage);
}

function renderTenants() {
  const container = document.getElementById("tenant-list");
  const count = document.getElementById("tenant-count");
  if (!container || !count) {
    return;
  }

  count.textContent = String(appState.tenants.length);
  if (appState.tenants.length === 0) {
    container.className = "stack empty";
    container.textContent = t("no_tenants");
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.tenants.map((item) => `
    <article class="detail-card ${item.tenant.id === selectedTenantId() ? "active" : ""}">
      <div class="line">
        <div>
          <strong>${escapeHTML(item.tenant.name)}</strong>
          <div class="muted mono">${escapeHTML(item.tenant.slug)}</div>
          <div class="muted mono">${escapeHTML(item.tenant.id)}</div>
        </div>
        <button class="button ghost" type="button" data-tenant-select="${item.tenant.id}">${escapeHTML(item.tenant.id === selectedTenantId() ? t("selected_tenant") : t("tenant_filter"))}</button>
      </div>
      <div class="mini-list">
        <span class="tag">${escapeHTML(t("products_metric"))} ${item.product_count}</span>
        <span class="tag">${escapeHTML(t("devices_metric"))} ${item.device_count}</span>
        <span class="tag">${escapeHTML(t("groups_metric"))} ${item.group_count}</span>
        <span class="tag">${escapeHTML(t("rules_metric"))} ${item.rule_count}</span>
        <span class="tag">${escapeHTML(t("configs_metric"))} ${item.config_profile_count}</span>
        <span class="tag">${escapeHTML(t("firmware_metric"))} ${item.firmware_count}</span>
        <span class="tag">${escapeHTML(t("ota_metric"))} ${item.ota_campaign_count}</span>
      </div>
      <div class="tag-list">${renderKVTags(item.tenant.metadata, t("no_metadata"))}</div>
    </article>
  `).join("");

  container.querySelectorAll("[data-tenant-select]").forEach((button) => {
    button.addEventListener("click", () => {
      const next = button.dataset.tenantSelect === selectedTenantId() ? "" : button.dataset.tenantSelect;
      saveTenantSelection(next);
      refreshAll().catch(handleGlobalError);
    });
  });
}

function renderFirmwareArtifacts() {
  const container = document.getElementById("firmware-list");
  const count = document.getElementById("firmware-count");
  if (!container || !count) {
    return;
  }

  count.textContent = String(appState.firmwareArtifacts.length);
  if (appState.firmwareArtifacts.length === 0) {
    container.className = "stack empty";
    container.textContent = t("no_firmware");
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.firmwareArtifacts.map((item) => `
    <article class="detail-card">
      <div class="line">
        <div>
          <strong>${escapeHTML(item.artifact.name)}</strong>
          <div class="muted mono">${escapeHTML(item.artifact.id)}</div>
          <div class="muted">${escapeHTML(item.artifact.notes || "")}</div>
        </div>
        <span class="chip">${escapeHTML(item.artifact.version)}</span>
      </div>
      <div class="mini-list">
        ${item.tenant ? `<span class="tag">${escapeHTML(item.tenant.name)}</span>` : ""}
        ${item.product ? `<span class="tag">${escapeHTML(item.product.name)}</span>` : `<span class="tag">${escapeHTML(t("all_products"))}</span>`}
        <span class="tag">${escapeHTML(t("updated"))} ${escapeHTML(formatTime(item.artifact.updated_at))}</span>
        <span class="tag">${escapeHTML(String(item.artifact.size_bytes || 0))} bytes</span>
      </div>
      <div class="tag-list">${renderKVTags(item.artifact.metadata, t("no_metadata"))}</div>
      <pre>${escapeHTML(pretty({
        file_name: item.artifact.file_name,
        url: item.artifact.url,
        checksum: item.artifact.checksum,
        checksum_type: item.artifact.checksum_type,
      }))}</pre>
    </article>
  `).join("");
}

function renderOTACampaigns() {
  const container = document.getElementById("ota-list");
  const count = document.getElementById("ota-count");
  if (!container || !count) {
    return;
  }

  count.textContent = String(appState.otaCampaigns.length);
  if (appState.otaCampaigns.length === 0) {
    container.className = "stack empty";
    container.textContent = t("no_ota");
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.otaCampaigns.map((item) => `
    <article class="detail-card">
      <div class="line">
        <div>
          <strong>${escapeHTML(item.campaign.name)}</strong>
          <div class="muted mono">${escapeHTML(item.campaign.id)}</div>
        </div>
        <span class="severity ${escapeHTML(item.campaign.status)}">${escapeHTML(item.campaign.status)}</span>
      </div>
      <div class="mini-list">
        ${item.tenant ? `<span class="tag">${escapeHTML(item.tenant.name)}</span>` : ""}
        ${item.firmware ? `<span class="tag">${escapeHTML(item.firmware.name)} ${escapeHTML(item.firmware.version)}</span>` : ""}
        ${item.product ? `<span class="tag">${escapeHTML(item.product.name)}</span>` : ""}
        ${item.group ? `<span class="tag">${escapeHTML(item.group.name)}</span>` : ""}
        ${item.device ? `<span class="tag">${escapeHTML(item.device.name)}</span>` : ""}
      </div>
      <div class="detail-meta-grid">
        <div class="meta-tile"><span>${escapeHTML(t("total_devices"))}</span><strong>${item.campaign.total_devices || 0}</strong></div>
        <div class="meta-tile"><span>${escapeHTML(t("dispatched"))}</span><strong>${item.campaign.dispatched_count || 0}</strong></div>
        <div class="meta-tile"><span>${escapeHTML(t("acked"))}</span><strong>${item.campaign.acked_count || 0}</strong></div>
        <div class="meta-tile"><span>${escapeHTML(t("failed"))}</span><strong>${item.campaign.failed_count || 0}</strong></div>
      </div>
      <div class="muted">${escapeHTML(formatTime(item.campaign.updated_at))}</div>
    </article>
  `).join("");
}

const originalSyncFormOptions = syncFormOptions;
syncFormOptions = function syncFormOptionsEnhanced() {
  originalSyncFormOptions();
  syncEnhancedSelects();
};

renderStats = function renderStatsEnhanced(metrics) {
  const container = document.getElementById("stats-grid");
  if (!container) {
    return;
  }

  const cards = [
    [t("tenants_metric"), appState.tenants.length],
    [t("products_metric"), appState.products.length],
    [t("devices_metric"), appState.devices.length],
    [t("online_metric"), metrics?.online_devices || 0],
    [t("groups_metric"), appState.groups.length],
    [t("rules_metric"), appState.rules.length],
    [t("configs_metric"), appState.configProfiles.length],
    [t("firmware_metric"), appState.firmwareArtifacts.length],
    [t("ota_metric"), appState.otaCampaigns.length],
    [t("alerts_metric"), appState.alerts.length],
    [t("telemetry_metric"), metrics?.telemetry_received || 0],
    [t("command_ack_metric"), metrics?.command_acks || 0],
    [t("http_ingest_metric"), metrics?.ingress?.http_ingest_accepted || 0],
    [t("mqtt_metric"), metrics?.ingress?.mqtt_messages_received || 0],
    [t("samples_metric"), metrics?.storage?.telemetry_samples || 0],
    [t("persist_errors_metric"), metrics?.storage?.persist_errors || 0],
  ];

  container.innerHTML = cards.map(([name, value]) => `
    <article class="metric-card">
      <span class="metric-label">${escapeHTML(name)}</span>
      <strong class="metric-value">${escapeHTML(String(value))}</strong>
    </article>
  `).join("");
};

renderRules = function renderRulesEnhanced() {
  const container = document.getElementById("rule-list");
  const count = document.getElementById("rule-count");
  if (!container || !count) {
    return;
  }

  count.textContent = String(appState.rules.length);
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
        ${item.tenant ? `<span class="tag">${escapeHTML(item.tenant.name)}</span>` : ""}
        ${item.product ? `<span class="tag">Product ${escapeHTML(item.product.name)}</span>` : ""}
        ${item.group ? `<span class="tag">Group ${escapeHTML(item.group.name)}</span>` : ""}
        ${item.device ? `<span class="tag">Device ${escapeHTML(item.device.name)}</span>` : ""}
        <span class="tag">Cooldown ${item.rule.cooldown_seconds || 0}s</span>
      </div>
      <div class="tag-list">
        ${(item.rule.actions || []).map((action) => {
          if (action.type === "send_command") {
            return `<span class="tag">send_command ${escapeHTML(action.name || "")}</span>`;
          }
          if (action.type === "apply_config_profile") {
            return `<span class="tag">apply_config_profile ${escapeHTML(action.config_profile_id || "")}</span>`;
          }
          return `<span class="tag">alert ${escapeHTML(action.severity || item.rule.severity || "")}</span>`;
        }).join("")}
      </div>
    </article>
  `).join("");
};

const originalApplyTranslations = applyTranslations;
applyTranslations = function applyTranslationsEnhanced() {
  originalApplyTranslations();

  setText("#tenant-form label:nth-of-type(1) span", t("tenant_name"));
  setText("#tenant-form label:nth-of-type(2) span", t("tenant_slug"));
  setText("#tenant-form label:nth-of-type(3) span", appState.locale === "zh" ? "描述" : "Description");
  setText("#tenant-form label:nth-of-type(4) span", appState.locale === "zh" ? "元数据 JSON" : "Metadata JSON");
  setText("#tenant-form button[type=\"submit\"]", t("create_tenant"));
  setText("#rule-form label:nth-of-type(11) span", t("action_type"));
  setText("#rule-action-command-wrap span", t("command_name"));
  setText("#rule-action-config-wrap span", appState.locale === "zh" ? "配置模板" : "Config Profile");
  setText("#rule-action-message-wrap span", t("alert_message"));
  setText("#rule-action-params-wrap span", t("command_params"));
  setText("#firmware-form .actions .button", t("create_firmware"));
  setText("#ota-form .actions .button", t("create_ota"));

  setText("[data-view=\"products\"] > .panel:nth-of-type(3) .section-kicker", appState.locale === "zh" ? "租户" : "Tenant");
  setText("[data-view=\"products\"] > .panel:nth-of-type(3) .section-head h3", t("tenant_workspace"));
  setText("[data-view=\"config\"] > .overview-grid .panel:nth-of-type(1) .section-kicker", appState.locale === "zh" ? "固件" : "Firmware");
  setText("[data-view=\"config\"] > .overview-grid .panel:nth-of-type(1) .section-head h3", t("firmware_repo"));
  setText("[data-view=\"config\"] > .overview-grid .panel:nth-of-type(2) .section-kicker", appState.locale === "zh" ? "OTA" : "OTA");
  setText("[data-view=\"config\"] > .overview-grid .panel:nth-of-type(2) .section-head h3", appState.locale === "zh" ? "OTA 任务" : "OTA Campaigns");

  setText("#firmware-form label:nth-of-type(1) span", t("firmware_name"));
  setText("#firmware-form label:nth-of-type(2) span", t("firmware_version"));
  setText("#firmware-form label:nth-of-type(3) span", appState.locale === "zh" ? "产品" : "Product");
  setText("#firmware-form label:nth-of-type(4) span", appState.locale === "zh" ? "文件名" : "File Name");
  setText("#firmware-form label:nth-of-type(5) span", t("firmware_url"));
  setText("#firmware-form label:nth-of-type(6) span", appState.locale === "zh" ? "校验和" : "Checksum");
  setText("#firmware-form label:nth-of-type(7) span", appState.locale === "zh" ? "校验算法" : "Checksum Type");
  setText("#firmware-form label:nth-of-type(8) span", appState.locale === "zh" ? "文件大小" : "Size Bytes");
  setText("#firmware-form label:nth-of-type(9) span", appState.locale === "zh" ? "备注" : "Notes");
  setText("#firmware-form label:nth-of-type(10) span", appState.locale === "zh" ? "元数据 JSON" : "Metadata JSON");
  setText("#ota-form label:nth-of-type(1) span", appState.locale === "zh" ? "任务名称" : "Campaign Name");
  setText("#ota-form label:nth-of-type(2) span", appState.locale === "zh" ? "固件" : "Firmware");
  setText("#ota-form label:nth-of-type(3) span", appState.locale === "zh" ? "产品范围" : "Product Scope");
  setText("#ota-form label:nth-of-type(4) span", appState.locale === "zh" ? "分组范围" : "Group Scope");
  setText("#ota-form label:nth-of-type(5) span", appState.locale === "zh" ? "设备范围" : "Device Scope");

  setPlaceholder("tenant-name", appState.locale === "zh" ? "华东工厂" : "factory-east");
  setPlaceholder("tenant-slug", appState.locale === "zh" ? "factory-east" : "factory-east");
  setPlaceholder("tenant-description", appState.locale === "zh" ? "华东工厂业务单元" : "Factory East business unit");
  setPlaceholder("rule-action-command", "reboot");
  setPlaceholder("rule-action-message", appState.locale === "zh" ? "温度超过阈值" : "Temperature is over threshold");
  setPlaceholder("firmware-name", appState.locale === "zh" ? "esp8266-通用固件" : "esp8266-universal");
  setPlaceholder("firmware-version", "1.0.0");
  setPlaceholder("firmware-url", "https://example.com/firmware.bin");
  setPlaceholder("ota-name", appState.locale === "zh" ? "华东工厂灰度发布" : "east-factory-rollout");

  const selector = document.getElementById("tenant-selector");
  if (selector) {
    selector.setAttribute("aria-label", t("tenant_filter"));
    selector.title = `${t("tenant_filter")}: ${selectedTenantName()}`;
  }
  syncTenantSelector();
  syncRuleActionVisibility();
};

refreshAll = async function refreshAllEnhanced() {
  const keepCurrentDetail = isEditingTextField();
  const [health, metrics, catalog, tenants] = await Promise.all([
    requestJSON("/healthz"),
    requestJSON("/metrics"),
    requestJSON("/api/v1/protocol-catalog"),
    requestJSON("/api/v1/tenants"),
  ]);

  appState.health = health;
  appState.metrics = metrics;
  appState.protocolCatalog = catalog;
  appState.tenants = tenants;

  if (selectedTenantId() && !appState.tenants.some((item) => item.tenant.id === selectedTenantId())) {
    saveTenantSelection("");
  }

  const [products, devices, groups, rules, alerts, configProfiles, firmwareArtifacts, otaCampaigns, simulators] = await Promise.all([
    requestJSON(withTenantQuery("/api/v1/products")),
    requestJSON(withTenantQuery("/api/v1/devices")),
    requestJSON(withTenantQuery("/api/v1/groups")),
    requestJSON(withTenantQuery("/api/v1/rules")),
    requestJSON(withTenantQuery("/api/v1/alerts", { limit: 20 })),
    requestJSON(withTenantQuery("/api/v1/config-profiles")),
    requestJSON(withTenantQuery("/api/v1/firmware")),
    requestJSON(withTenantQuery("/api/v1/ota-campaigns")),
    requestJSON("/api/v1/simulators"),
  ]);

  appState.products = products;
  appState.devices = devices;
  appState.groups = groups;
  appState.rules = rules;
  appState.alerts = alerts;
  appState.configProfiles = configProfiles;
  appState.firmwareArtifacts = firmwareArtifacts;
  appState.otaCampaigns = otaCampaigns;
  appState.simulators = selectedTenantId()
    ? simulators.filter((item) => item.device.tenant_id === selectedTenantId())
    : simulators;

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
  renderProducts();
  renderProtocolCatalog();
  renderDevices();
  renderGroups();
  renderRules();
  renderAlerts();
  renderConfigProfiles();
  renderSimulators();
  renderTenants();
  renderFirmwareArtifacts();
  renderOTACampaigns();
  applyTranslations();

  if (!keepCurrentDetail) {
    await refreshSelectedDevice();
  }
};

function buildRuleActionsFromForm() {
  const type = document.getElementById("rule-action-type").value;
  if (type === "send_command") {
    const name = document.getElementById("rule-action-command").value.trim();
    if (!name) {
      throw new Error(t("command_name"));
    }
    return [{
      type,
      name,
      params: parseJSON(document.getElementById("rule-action-params").value, {}),
    }];
  }
  if (type === "apply_config_profile") {
    const configProfileID = document.getElementById("rule-action-config-id").value;
    if (!configProfileID) {
      throw new Error(t("select_profile_first"));
    }
    return [{
      type,
      config_profile_id: configProfileID,
    }];
  }
  return [{
    type: "alert",
    severity: document.getElementById("rule-severity").value,
    message: document.getElementById("rule-action-message").value.trim(),
  }];
}

function bindCaptureSubmit(id, handler) {
  const form = document.getElementById(id);
  if (!form) {
    return;
  }
  form.addEventListener("submit", (event) => {
    event.preventDefault();
    event.stopImmediatePropagation();
    handler().catch((error) => {
      console.error(error);
    });
  }, true);
}

function bindEnhancedForms() {
  bindCaptureSubmit("product-form", async () => {
    try {
      setHint("product-status", t("create_in_progress"));
      await requestJSON("/api/v1/products", {
        method: "POST",
        body: JSON.stringify({
          tenant_id: selectedTenantId(),
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

  bindCaptureSubmit("device-form", async () => {
    try {
      setHint("device-status", t("create_in_progress"));
      await requestJSON("/api/v1/devices", {
        method: "POST",
        body: JSON.stringify({
          tenant_id: selectedTenantId(),
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

  bindCaptureSubmit("group-form", async () => {
    try {
      setHint("group-status", t("create_in_progress"));
      await requestJSON("/api/v1/groups", {
        method: "POST",
        body: JSON.stringify({
          tenant_id: selectedTenantId(),
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

  bindCaptureSubmit("rule-form", async () => {
    try {
      setHint("rule-status", t("create_in_progress"));
      await requestJSON("/api/v1/rules", {
        method: "POST",
        body: JSON.stringify({
          tenant_id: selectedTenantId(),
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
          actions: buildRuleActionsFromForm(),
        }),
      });
      document.getElementById("rule-name").value = "";
      document.getElementById("rule-description").value = "";
      document.getElementById("rule-action-message").value = "";
      setHint("rule-status", t("created_ok"));
      await refreshAll();
    } catch (error) {
      setHint("rule-status", error.message, true);
    }
  });

  bindCaptureSubmit("config-form", async () => {
    try {
      setHint("config-status", t("create_in_progress"));
      await requestJSON("/api/v1/config-profiles", {
        method: "POST",
        body: JSON.stringify({
          tenant_id: selectedTenantId(),
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

  bindCaptureSubmit("tenant-form", async () => {
    try {
      setHint("tenant-status", t("create_in_progress"));
      const tenant = await requestJSON("/api/v1/tenants", {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("tenant-name").value.trim(),
          slug: document.getElementById("tenant-slug").value.trim(),
          description: document.getElementById("tenant-description").value.trim(),
          metadata: parseJSON(document.getElementById("tenant-metadata").value, {}),
        }),
      });
      document.getElementById("tenant-name").value = "";
      document.getElementById("tenant-slug").value = "";
      document.getElementById("tenant-description").value = "";
      saveTenantSelection(tenant.id || "");
      setHint("tenant-status", t("created_ok"));
      await refreshAll();
    } catch (error) {
      setHint("tenant-status", error.message, true);
    }
  });

  bindCaptureSubmit("firmware-form", async () => {
    try {
      setHint("firmware-status", t("create_in_progress"));
      await requestJSON("/api/v1/firmware", {
        method: "POST",
        body: JSON.stringify({
          tenant_id: selectedTenantId(),
          product_id: document.getElementById("firmware-product-id").value,
          name: document.getElementById("firmware-name").value.trim(),
          version: document.getElementById("firmware-version").value.trim(),
          file_name: document.getElementById("firmware-file-name").value.trim(),
          url: document.getElementById("firmware-url").value.trim(),
          checksum: document.getElementById("firmware-checksum").value.trim(),
          checksum_type: document.getElementById("firmware-checksum-type").value.trim(),
          size_bytes: Number(document.getElementById("firmware-size-bytes").value || 0),
          metadata: parseJSON(document.getElementById("firmware-metadata").value, {}),
          notes: document.getElementById("firmware-notes").value.trim(),
        }),
      });
      document.getElementById("firmware-name").value = "";
      document.getElementById("firmware-version").value = "";
      document.getElementById("firmware-file-name").value = "";
      document.getElementById("firmware-url").value = "";
      setHint("firmware-status", t("created_ok"));
      await refreshAll();
    } catch (error) {
      setHint("firmware-status", error.message, true);
    }
  });

  bindCaptureSubmit("ota-form", async () => {
    try {
      setHint("ota-status", t("create_in_progress"));
      await requestJSON("/api/v1/ota-campaigns", {
        method: "POST",
        body: JSON.stringify({
          tenant_id: selectedTenantId(),
          name: document.getElementById("ota-name").value.trim(),
          firmware_id: document.getElementById("ota-firmware-id").value,
          product_id: document.getElementById("ota-product-id").value,
          group_id: document.getElementById("ota-group-id").value,
          device_id: document.getElementById("ota-device-id").value,
        }),
      });
      document.getElementById("ota-name").value = "";
      setHint("ota-status", t("created_ok"));
      await refreshAll();
    } catch (error) {
      setHint("ota-status", error.message, true);
    }
  });

  document.getElementById("tenant-selector")?.addEventListener("change", (event) => {
    saveTenantSelection(event.target.value);
    refreshAll().catch(handleGlobalError);
  });
  document.getElementById("locale-toggle")?.addEventListener("click", () => {
    window.setTimeout(() => {
      syncEnhancedSelects();
      renderTenants();
      renderFirmwareArtifacts();
      renderOTACampaigns();
      applyTranslations();
    }, 0);
  });
  document.getElementById("rule-action-type")?.addEventListener("change", syncRuleActionVisibility);
  document.getElementById("ota-group-id")?.addEventListener("change", (event) => {
    const group = getGroup(event.target.value);
    if (group?.product) {
      document.getElementById("ota-product-id").value = group.product.id;
    }
  });
  document.getElementById("ota-device-id")?.addEventListener("change", (event) => {
    const device = getDevice(event.target.value);
    if (device?.product) {
      document.getElementById("ota-product-id").value = device.product.id;
    }
  });
}

function bootstrapAdvancedConsole() {
  bindEnhancedForms();
  syncEnhancedSelects();
  applyTranslations();
  refreshAll().catch(handleGlobalError);
}

bootstrapAdvancedConsole();
