const appState = {
  devices: [],
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
  const payload = contentType.includes("application/json")
    ? await response.json()
    : await response.text();

  if (!response.ok) {
    const message = typeof payload === "string" ? payload : (payload.error || response.statusText);
    throw new Error(message);
  }
  return payload;
}

function parseJSON(text, fallback) {
  const value = (text || "").trim();
  if (!value) {
    return fallback;
  }
  return JSON.parse(value);
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function formatTime(value) {
  if (!value) {
    return "-";
  }
  return new Date(value).toLocaleString();
}

function pretty(value) {
  return JSON.stringify(value ?? {}, null, 2);
}

function isEditingTextField() {
  const element = document.activeElement;
  if (!element) {
    return false;
  }

  const tagName = element.tagName;
  return tagName === "INPUT" || tagName === "TEXTAREA";
}

function setHint(id, message, isError = false) {
  const node = document.getElementById(id);
  if (!node) {
    return;
  }
  node.textContent = message;
  node.style.color = isError ? "#b42318" : "";
}

function renderHealth(health) {
  const dot = document.getElementById("health-dot");
  const text = document.getElementById("health-text");
  const healthy = health.status === "ok";
  dot.classList.toggle("online", healthy);
  text.textContent = healthy
    ? `运行正常 · ${new Date(health.time).toLocaleTimeString()}`
    : "运行异常";
}

function renderStats(stats) {
  const data = [
    ["已注册设备", stats.registered_devices],
    ["在线设备", stats.online_devices],
    ["累计连接", stats.total_connections],
    ["拒绝连接", stats.rejected_connections],
    ["遥测总量", stats.telemetry_received],
    ["命令下发", stats.commands_sent],
    ["命令回执", stats.command_acks],
  ];

  document.getElementById("stats-grid").innerHTML = data.map(([name, value]) => `
    <article class="metric-card">
      <span class="metric-label">${name}</span>
      <strong class="metric-value">${value ?? 0}</strong>
    </article>
  `).join("");
}

function renderDevices() {
  const container = document.getElementById("device-list");
  document.getElementById("device-count").textContent = `${appState.devices.length}`;

  if (appState.devices.length === 0) {
    container.className = "stack empty";
    container.textContent = "暂无设备";
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.devices.map((item) => `
    <article class="device-card ${item.device.id === appState.selectedDeviceId ? "active" : ""}">
      <button data-device-id="${item.device.id}">
        <div class="line">
          <strong>${escapeHTML(item.device.name)}</strong>
          <span class="pill ${item.online ? "online" : "offline"}">${item.online ? "online" : "offline"}</span>
        </div>
        <div class="muted mono">${escapeHTML(item.device.id)}</div>
        <div class="muted">最后活跃 ${formatTime(item.last_seen)}</div>
        <div class="muted mono">Token ${escapeHTML(item.device.token || "")}</div>
      </button>
    </article>
  `).join("");

  container.querySelectorAll("[data-device-id]").forEach((button) => {
    button.addEventListener("click", async () => {
      appState.selectedDeviceId = button.dataset.deviceId;
      renderDevices();
      await refreshSelectedDevice();
    });
  });
}

async function refreshSelectedDevice() {
  const panel = document.getElementById("device-detail");
  if (!appState.selectedDeviceId) {
    document.getElementById("selected-device-name").textContent = "未选择";
    panel.className = "stack empty";
    panel.textContent = "选择一个设备查看详情。";
    return;
  }

  const [device, telemetry, commands] = await Promise.all([
    requestJSON(`/api/v1/devices/${appState.selectedDeviceId}`),
    requestJSON(`/api/v1/devices/${appState.selectedDeviceId}/telemetry?limit=20`),
    requestJSON(`/api/v1/devices/${appState.selectedDeviceId}/commands?limit=20`),
  ]);

  document.getElementById("selected-device-name").textContent = device.device.name;
  panel.className = "device-detail-grid";
  panel.innerHTML = `
    <article class="detail-card">
      <div class="line">
        <strong>${escapeHTML(device.device.name)}</strong>
        <span class="pill ${device.online ? "online" : "offline"}">${device.online ? "online" : "offline"}</span>
      </div>
      <div class="detail-meta">
        <div class="muted mono">${escapeHTML(device.device.id)}</div>
        <div class="detail-meta-grid">
          <div class="meta-tile">
            <span>创建时间</span>
            <strong>${formatTime(device.device.created_at)}</strong>
          </div>
          <div class="meta-tile">
            <span>连接时间</span>
            <strong>${formatTime(device.connected_at)}</strong>
          </div>
          <div class="meta-tile">
            <span>最后活跃</span>
            <strong>${formatTime(device.last_seen)}</strong>
          </div>
        </div>
      </div>
      <pre>${escapeHTML(pretty(device.device.metadata))}</pre>
    </article>
    <article class="detail-card">
      <div class="line"><strong>发送命令</strong></div>
      <form id="command-form" class="grid-form">
        <label>
          <span>命令名</span>
          <input id="command-name" type="text" value="reboot">
        </label>
        <label class="wide">
          <span>参数 JSON</span>
          <textarea id="command-params" rows="4">{"delay":1}</textarea>
        </label>
        <div class="actions wide">
          <button class="button primary" type="submit">发送命令</button>
          <span id="command-status" class="hint"></span>
        </div>
      </form>
    </article>
    <article class="detail-card">
      <div class="line"><strong>最近遥测</strong></div>
      <pre>${escapeHTML(pretty(telemetry))}</pre>
    </article>
    <article class="detail-card">
      <div class="line"><strong>最近命令</strong></div>
      <pre>${escapeHTML(pretty(commands))}</pre>
    </article>
  `;

  document.getElementById("command-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("command-status", "发送中...");
      await requestJSON(`/api/v1/devices/${appState.selectedDeviceId}/commands`, {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("command-name").value.trim(),
          params: parseJSON(document.getElementById("command-params").value, {}),
        }),
      });
      setHint("command-status", "命令已提交");
      await refreshAll();
    } catch (error) {
      setHint("command-status", error.message, true);
    }
  });
}

function renderSimulators() {
  const container = document.getElementById("sim-list");
  if (appState.simulators.length === 0) {
    container.className = "stack empty";
    container.textContent = "暂无模拟器";
    return;
  }

  container.className = "stack";
  container.innerHTML = appState.simulators.map((sim) => `
    <article class="sim-card">
      <div class="line">
        <div>
          <strong>${escapeHTML(sim.device.name)}</strong>
          <div class="muted mono">${escapeHTML(sim.device.id)}</div>
          <div class="muted mono">${escapeHTML(sim.id)}</div>
        </div>
        <div class="sim-summary">
          <span class="pill ${sim.connected ? "online" : "offline"}">${sim.connected ? "connected" : "disconnected"}</span>
          <span class="pill">ack ${sim.auto_ack ? "on" : "off"}</span>
          <span class="pill">ping ${sim.auto_ping ? "on" : "off"}</span>
          <span class="pill">telemetry ${sim.auto_telemetry ? `${sim.telemetry_interval_ms}ms` : "manual"}</span>
        </div>
      </div>
      <div class="sim-actions">
        <button class="button ghost" data-connect="${sim.id}">连接</button>
        <button class="button ghost" data-disconnect="${sim.id}">断开</button>
        <button class="button accent" data-telemetry="${sim.id}">发送遥测</button>
        <button class="button ghost" data-remove="${sim.id}">删除</button>
      </div>
      <div class="sim-grid">
        <div class="detail-card">
          <div class="line"><strong>配置</strong></div>
          <pre>${escapeHTML(pretty({
            auto_ack: sim.auto_ack,
            auto_ping: sim.auto_ping,
            auto_telemetry: sim.auto_telemetry,
            telemetry_interval_ms: sim.telemetry_interval_ms,
            default_values: sim.default_values,
          }))}</pre>
        </div>
        <div class="detail-card">
          <div class="line"><strong>状态</strong></div>
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
          <div class="line"><strong>日志</strong></div>
          <pre>${escapeHTML((sim.logs || []).map((entry) => `[${formatTime(entry.timestamp)}] ${entry.level.toUpperCase()} ${entry.message}`).join("\n"))}</pre>
        </div>
      </div>
    </article>
  `).join("");

  bindSimulatorButtons();
}

function bindSimulatorButtons() {
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
      await requestJSON(`/api/v1/simulators/${id}/connect`, { method: "POST", body: "{}" });
    } else if (action === "disconnect") {
      await requestJSON(`/api/v1/simulators/${id}/disconnect`, { method: "POST", body: "{}" });
    } else if (action === "remove") {
      await requestJSON(`/api/v1/simulators/${id}`, { method: "DELETE" });
    } else if (action === "telemetry") {
      const sim = appState.simulators.find((item) => item.id === id);
      await requestJSON(`/api/v1/simulators/${id}/telemetry`, {
        method: "POST",
        body: JSON.stringify({ values: sim ? sim.default_values : {} }),
      });
    }
    await refreshAll();
  } catch (error) {
    window.alert(error.message);
  }
}

async function refreshAll() {
  const keepCurrentDetail = isEditingTextField();
  const [health, stats, devices, simulators] = await Promise.all([
    requestJSON("/healthz"),
    requestJSON("/metrics"),
    requestJSON("/api/v1/devices"),
    requestJSON("/api/v1/simulators"),
  ]);

  appState.devices = devices;
  appState.simulators = simulators;
  if (!appState.selectedDeviceId && devices.length > 0) {
    appState.selectedDeviceId = devices[0].device.id;
  }
  if (appState.selectedDeviceId && !devices.some((item) => item.device.id === appState.selectedDeviceId)) {
    appState.selectedDeviceId = devices.length > 0 ? devices[0].device.id : "";
  }

  renderHealth(health);
  renderStats(stats);
  renderDevices();
  renderSimulators();
  if (!keepCurrentDetail) {
    await refreshSelectedDevice();
  }
}

function bindForms() {
  document.getElementById("device-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("device-status", "创建中...");
      await requestJSON("/api/v1/devices", {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("device-name").value.trim(),
          metadata: parseJSON(document.getElementById("device-metadata").value, {}),
        }),
      });
      document.getElementById("device-name").value = "";
      setHint("device-status", "设备已创建");
      await refreshAll();
    } catch (error) {
      setHint("device-status", error.message, true);
    }
  });

  document.getElementById("sim-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setHint("sim-status", "创建中...");
      await requestJSON("/api/v1/simulators", {
        method: "POST",
        body: JSON.stringify({
          name: document.getElementById("sim-name").value.trim(),
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
      setHint("sim-status", "模拟器已创建");
      await refreshAll();
    } catch (error) {
      setHint("sim-status", error.message, true);
    }
  });

  document.getElementById("refresh-button").addEventListener("click", refreshAll);
}

async function bootstrap() {
  bindForms();
  await refreshAll();
  window.setInterval(refreshAll, 3000);
}

bootstrap().catch((error) => {
  document.getElementById("health-text").textContent = error.message;
});
