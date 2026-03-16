const VIEW_TITLES = {
  overview: "Overview",
  products: "Product Center",
  devices: "Device Center",
  governance: "Governance",
  config: "Config Center",
  simulator: "Simulator Lab",
};

const appState = {
  currentView: "overview",
  health: null,
  metrics: null,
  products: [],
  devices: [],
  groups: [],
  rules: [],
  alerts: [],
  configProfiles: [],
  simulators: [],
  selectedDeviceId: "",
};

async function requestJSON(path, options = {}) {
  const response = await fetch(path, {
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
  return value ? new Date(value).toLocaleString("zh-CN") : "-";
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
  text.textContent = healthy ? `Runtime healthy | ${formatTime(health.time)}` : "Runtime unavailable";
}

function renderStats(metrics) {
  const container = document.getElementById("stats-grid");
  if (!container) {
    return;
  }

  const cards = [
    ["Products", appState.products.length],
    ["Devices", metrics?.registered_devices || 0],
    ["Online", metrics?.online_devices || 0],
    ["Groups", appState.groups.length],
    ["Rules", appState.rules.length],
    ["Alerts", appState.alerts.length],
    ["Configs", appState.configProfiles.length],
    ["Connections", metrics?.total_connections || 0],
    ["Telemetry", metrics?.telemetry_received || 0],
    ["Command Ack", metrics?.command_acks || 0],
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
    deviceContainer.textContent = "No devices yet.";
  } else {
    deviceContainer.className = "stack";
    deviceContainer.innerHTML = recentDevices.map((item) => `
      <article class="detail-card">
        <div class="line">
          <div>
            <strong>${escapeHTML(item.device.name)}</strong>
            <div class="muted mono">${escapeHTML(item.device.id)}</div>
          </div>
          <span class="pill ${item.online ? "online" : "offline"}">${item.online ? "online" : "offline"}</span>
        </div>
        <div class="mini-list">
          ${item.product ? `<span class="tag">${escapeHTML(item.product.name)}</span>` : '<span class="tag">Unbound</span>'}
          ${(item.groups || []).map((group) => `<span class="tag">${escapeHTML(group.name)}</span>`).join("")}
        </div>
        <div class="list-actions">
          <button class="button ghost" type="button" data-overview-device="${item.device.id}">Inspect Device</button>
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
    alertContainer.textContent = "No alerts yet.";
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
  syncSelect("device-product-id", productOptions, "Unbound");
  syncSelect("sim-product-id", productOptions, "Unbound");
  syncSelect("group-product-id", productOptions, "Any product");
  syncSelect("rule-product-id", productOptions, "Auto");
  syncSelect("config-product-id", productOptions, "Optional");
  syncSelect("rule-group-id", appState.groups.map((item) => ({ value: item.group.id, label: item.group.name })), "Optional");
  syncSelect("rule-device-id", appState.devices.map((item) => ({ value: item.device.id, label: item.device.name })), "Optional");
}

function renderProducts() {
  const container = document.getElementById("product-list");
  document.getElementById("product-count").textContent = `${appState.products.length}`;
  if (appState.products.length === 0) {
    container.className = "stack empty";
    container.textContent = "No products yet.";
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
        <div class="meta-tile"><span>Online</span><strong>${item.online_count}</strong></div>
        <div class="meta-tile"><span>Properties</span><strong>${(item.product.thing_model.properties || []).length}</strong></div>
        <div class="meta-tile"><span>Services</span><strong>${(item.product.thing_model.services || []).length}</strong></div>
        <div class="meta-tile"><span>Version</span><strong>${item.product.thing_model.version || 0}</strong></div>
      </div>
      <div class="tag-list">${renderKVTags(item.product.metadata, "No metadata")}</div>
      <pre>${escapeHTML(pretty(item.product.thing_model))}</pre>
    </article>
  `).join("");
}

function renderDevices() {
  const container = document.getElementById("device-list");
  document.getElementById("device-count").textContent = `${appState.devices.length}`;
  if (appState.devices.length === 0) {
    container.className = "stack empty";
    container.textContent = "No devices yet.";
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.devices.map((item) => `
    <article class="device-card ${item.device.id === appState.selectedDeviceId ? "active" : ""}">
      <button type="button" data-device-id="${item.device.id}">
        <div class="line">
          <strong>${escapeHTML(item.device.name)}</strong>
          <span class="pill ${item.online ? "online" : "offline"}">${item.online ? "online" : "offline"}</span>
        </div>
        <div class="muted mono">${escapeHTML(item.device.id)}</div>
        <div class="muted">${item.product ? `Product ${escapeHTML(item.product.name)}` : "Unbound"}</div>
        <div class="mini-list">${renderKVTags(item.device.tags, "No tags")}</div>
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
    container.textContent = "No groups yet.";
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
            <div class="muted">${item.product ? `Bound to ${escapeHTML(item.product.name)}` : "Any product"}</div>
          </div>
          <span class="chip">${item.device_count} devices</span>
        </div>
        <div class="detail-meta-grid">
          <div class="meta-tile"><span>Online</span><strong>${item.online_count}</strong></div>
          <div class="meta-tile"><span>Description</span><strong>${escapeHTML(item.group.description || "-")}</strong></div>
        </div>
        <div class="tag-list">${renderKVTags(item.group.tags, "No tags")}</div>
        <div class="list-actions">
          ${selectedDevice
            ? `<button class="button ghost" type="button" data-group-${member ? "remove" : "add"}="${item.group.id}">${member ? "Remove selected device" : "Add selected device"}</button>`
            : '<span class="subtle">Select a device to manage membership.</span>'}
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
    container.textContent = "No rules yet.";
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
    container.textContent = "No alerts yet.";
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
        ${item.status !== "acknowledged" && item.status !== "resolved" ? `<button class="button ghost" type="button" data-alert-ack="${item.id}">Acknowledge</button>` : ""}
        ${item.status !== "resolved" ? `<button class="button primary" type="button" data-alert-resolve="${item.id}">Resolve</button>` : ""}
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
    container.textContent = "No config profiles yet.";
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
          ${item.product ? `<span class="tag">Product ${escapeHTML(item.product.name)}</span>` : '<span class="tag">All products</span>'}
          <span class="tag">Updated ${escapeHTML(formatTime(item.profile.updated_at))}</span>
        </div>
        <div class="list-actions">
          ${canApply ? `<button class="button primary" type="button" data-config-apply="${item.profile.id}">Apply To Selected Device</button>` : ""}
          ${!selectedDevice ? '<span class="subtle">Select a device to apply.</span>' : ""}
          ${scopedToAnotherProduct ? '<span class="subtle">Selected device product does not match this profile.</span>' : ""}
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
    selectedName.textContent = "Unselected";
    panel.className = "stack empty";
    panel.textContent = "Select a device to inspect tags, shadow, commands and alerts.";
    return;
  }

  const [device, shadow, telemetry, commands, alerts] = await Promise.all([
    requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}`),
    requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/shadow`),
    requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/telemetry?limit=20`),
    requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/commands?limit=20`),
    requestJSON(`/api/v1/alerts?device_id=${encodeURIComponent(appState.selectedDeviceId)}&limit=10`),
  ]);

  selectedName.textContent = device.device.name;
  panel.className = "device-detail-grid";
  panel.innerHTML = `
    <article class="detail-card">
      <div class="line">
        <div>
          <strong>${escapeHTML(device.device.name)}</strong>
          <div class="muted mono">${escapeHTML(device.device.id)}</div>
        </div>
        <span class="pill ${device.online ? "online" : "offline"}">${device.online ? "online" : "offline"}</span>
      </div>
      <div class="mini-list">
        ${device.product ? `<span class="tag">Product ${escapeHTML(device.product.name)}</span>` : '<span class="tag">Unbound</span>'}
        ${(device.groups || []).map((group) => `<span class="tag">${escapeHTML(group.name)}</span>`).join("")}
      </div>
      <div class="detail-meta-grid">
        <div class="meta-tile"><span>Connected</span><strong>${formatTime(device.connected_at)}</strong></div>
        <div class="meta-tile"><span>Last Seen</span><strong>${formatTime(device.last_seen)}</strong></div>
        <div class="meta-tile"><span>Token</span><strong class="mono">${escapeHTML(device.device.token || "-")}</strong></div>
      </div>
      <div class="tag-list">${renderKVTags(device.device.tags, "No tags")}</div>
      <pre>${escapeHTML(pretty(device.device.metadata || {}))}</pre>
    </article>

    <article class="detail-card">
      <div class="line"><strong>Device Tags</strong></div>
      <form id="device-tags-form" class="grid-form">
        <label class="wide">
          <span>Tags JSON</span>
          <textarea id="device-tags-editor" rows="5">${escapeHTML(pretty(device.device.tags || {}))}</textarea>
        </label>
        <div class="actions wide">
          <button class="button ghost" type="submit">Update Tags</button>
          <span id="device-tags-status" class="hint"></span>
        </div>
      </form>
    </article>

    <article class="detail-card">
      <div class="line"><strong>Remote Config</strong></div>
      <form id="device-config-form" class="grid-form">
        <label class="wide">
          <span>Config Profile</span>
          <select id="device-config-profile-id">
            <option value="">Select profile</option>
            ${appState.configProfiles.map((item) => `<option value="${item.profile.id}">${escapeHTML(item.profile.name)}${item.product ? ` | ${escapeHTML(item.product.name)}` : ""}</option>`).join("")}
          </select>
        </label>
        <div class="actions wide">
          <button class="button primary" type="submit">Apply Config Profile</button>
          <span id="device-config-status" class="hint"></span>
        </div>
      </form>
      <pre>${escapeHTML(pretty(shadow.desired || {}))}</pre>
    </article>

    <article class="detail-card">
      <div class="line"><strong>Device Shadow</strong></div>
      <form id="shadow-form" class="grid-form">
        <label class="wide">
          <span>Desired JSON</span>
          <textarea id="shadow-desired" rows="6">${escapeHTML(pretty(shadow.desired || {}))}</textarea>
        </label>
        <div class="actions wide">
          <button class="button ghost" type="submit">Update Desired</button>
          <span id="shadow-status" class="hint"></span>
        </div>
      </form>
      <pre>${escapeHTML(pretty({ reported: shadow.reported || {}, desired: shadow.desired || {}, updated_at: shadow.updated_at }))}</pre>
    </article>

    <article class="detail-card">
      <div class="line"><strong>Send Command</strong></div>
      <form id="command-form" class="grid-form">
        <label>
          <span>Command</span>
          <input id="command-name" type="text" value="reboot">
        </label>
        <label class="wide">
          <span>Params JSON</span>
          <textarea id="command-params" rows="4">{"delay":1}</textarea>
        </label>
        <div class="actions wide">
          <button class="button primary" type="submit">Send Command</button>
          <span id="command-status" class="hint"></span>
        </div>
      </form>
    </article>

    <article class="detail-card">
      <div class="line"><strong>Recent Telemetry</strong></div>
      <pre>${escapeHTML(pretty(telemetry))}</pre>
    </article>

    <article class="detail-card">
      <div class="line"><strong>Recent Commands</strong></div>
      <pre>${escapeHTML(pretty(commands))}</pre>
    </article>

    <article class="detail-card">
      <div class="line"><strong>Device Alerts</strong></div>
      <pre>${escapeHTML(pretty(alerts))}</pre>
    </article>
  `;

  document.getElementById("device-tags-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("device-tags-status", "Updating...");
      await requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/tags`, {
        method: "PUT",
        body: JSON.stringify({ tags: parseJSON(document.getElementById("device-tags-editor").value, {}) }),
      });
      setHint("device-tags-status", "Tags updated");
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
        throw new Error("select a config profile first");
      }
      setHint("device-config-status", "Applying...");
      await applyConfigProfile(profileID, appState.selectedDeviceId, "device-config-status");
    } catch (error) {
      setHint("device-config-status", error.message, true);
    }
  });

  document.getElementById("shadow-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("shadow-status", "Updating...");
      await requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/shadow`, {
        method: "PUT",
        body: JSON.stringify({ desired: parseJSON(document.getElementById("shadow-desired").value, {}) }),
      });
      setHint("shadow-status", "Shadow updated");
      await refreshAll();
    } catch (error) {
      setHint("shadow-status", error.message, true);
    }
  });

  document.getElementById("command-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("command-status", "Sending...");
      await requestJSON(`/api/v1/devices/${encodeURIComponent(appState.selectedDeviceId)}/commands`, {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("command-name").value.trim(),
          params: parseJSON(document.getElementById("command-params").value, {}),
        }),
      });
      setHint("command-status", "Command accepted");
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
    container.textContent = "No simulators yet.";
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
          <span class="pill ${sim.connected ? "online" : "offline"}">${sim.connected ? "connected" : "disconnected"}</span>
          <span class="pill">ack ${sim.auto_ack ? "on" : "off"}</span>
          <span class="pill">ping ${sim.auto_ping ? "on" : "off"}</span>
          <span class="pill">telemetry ${sim.auto_telemetry ? `${sim.telemetry_interval_ms}ms` : "manual"}</span>
        </div>
      </div>
      <div class="sim-actions">
        <button class="button ghost" type="button" data-connect="${sim.id}">Connect</button>
        <button class="button ghost" type="button" data-disconnect="${sim.id}">Disconnect</button>
        <button class="button accent" type="button" data-telemetry="${sim.id}">Send Telemetry</button>
        <button class="button ghost" type="button" data-remove="${sim.id}">Remove</button>
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
          <pre>${escapeHTML((sim.logs || []).map((entry) => `[${formatTime(entry.timestamp)}] ${entry.level.toUpperCase()} ${entry.message}`).join("\n") || "No logs yet.")}</pre>
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

  setHint(hintId, "Applying...");
  await requestJSON(`/api/v1/config-profiles/${encodeURIComponent(profileId)}/apply`, {
    method: "POST",
    body: JSON.stringify({ device_id: deviceId }),
  });
  setHint(hintId, "Config applied");
  await refreshAll();
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
  const target = VIEW_TITLES[viewId] ? viewId : "overview";
  appState.currentView = target;

  document.querySelectorAll("[data-view-target]").forEach((node) => {
    node.classList.toggle("active", node.dataset.viewTarget === target);
  });
  document.querySelectorAll("[data-view]").forEach((node) => {
    node.classList.toggle("active", node.dataset.view === target);
  });

  const title = document.getElementById("view-title");
  if (title) {
    title.textContent = VIEW_TITLES[target];
  }
}

async function refreshAll() {
  const keepCurrentDetail = isEditingTextField();
  const [health, metrics, products, devices, groups, rules, alerts, configProfiles, simulators] = await Promise.all([
    requestJSON("/healthz"),
    requestJSON("/metrics"),
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
  renderProducts();
  renderDevices();
  renderGroups();
  renderRules();
  renderAlerts();
  renderConfigProfiles();
  renderSimulators();

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
      setHint("product-status", "Creating...");
      await requestJSON("/api/v1/products", {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("product-name").value.trim(),
          description: document.getElementById("product-description").value.trim(),
          metadata: parseJSON(document.getElementById("product-metadata").value, {}),
          thing_model: parseJSON(document.getElementById("product-thing-model").value, {}),
        }),
      });
      document.getElementById("product-name").value = "";
      document.getElementById("product-description").value = "";
      setHint("product-status", "Product created");
      await refreshAll();
    } catch (error) {
      setHint("product-status", error.message, true);
    }
  });

  document.getElementById("device-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("device-status", "Creating...");
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
      setHint("device-status", "Device created");
      await refreshAll();
    } catch (error) {
      setHint("device-status", error.message, true);
    }
  });

  document.getElementById("group-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("group-status", "Creating...");
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
      setHint("group-status", "Group created");
      await refreshAll();
    } catch (error) {
      setHint("group-status", error.message, true);
    }
  });

  document.getElementById("rule-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("rule-status", "Creating...");
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
      setHint("rule-status", "Rule created");
      await refreshAll();
    } catch (error) {
      setHint("rule-status", error.message, true);
    }
  });

  document.getElementById("config-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("config-status", "Creating...");
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
      setHint("config-status", "Config profile created");
      await refreshAll();
    } catch (error) {
      setHint("config-status", error.message, true);
    }
  });

  document.getElementById("sim-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("sim-status", "Creating...");
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
      setHint("sim-status", "Simulator created");
      await refreshAll();
    } catch (error) {
      setHint("sim-status", error.message, true);
    }
  });

  document.getElementById("refresh-button").addEventListener("click", () => {
    refreshAll().catch(handleGlobalError);
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
  bindNavigation();
  bindForms();
  activateView(appState.currentView);
  await refreshAll();
  window.setInterval(() => {
    refreshAll().catch(handleGlobalError);
  }, 4000);
}

bootstrap().catch(handleGlobalError);
