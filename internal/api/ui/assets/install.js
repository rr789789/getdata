function readRuntimeConfig() {
  const config = window.__MVP_RUNTIME_CONFIG__ || {};
  return typeof config === "object" && config !== null ? config : {};
}

const installRuntime = readRuntimeConfig();
const installAPIBaseURL = String(installRuntime.api_base_url || "").trim().replace(/\/+$/, "");

function resolveInstallAPI(path) {
  const raw = String(path || "").trim();
  if (!raw) {
    return raw;
  }
  if (/^[a-z]+:\/\//i.test(raw)) {
    return raw;
  }
  if (!installAPIBaseURL) {
    return raw;
  }
  return raw.startsWith("/") ? `${installAPIBaseURL}${raw}` : `${installAPIBaseURL}/${raw}`;
}

async function installRequest(path, options = {}) {
  const response = await fetch(resolveInstallAPI(path), {
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
    ...options,
  });

  const contentType = response.headers.get("content-type") || "";
  const payload = contentType.includes("application/json") ? await response.json() : await response.text();
  if (!response.ok) {
    const message = typeof payload === "string" ? payload : (payload.error || response.statusText);
    throw new Error(message);
  }
  return payload;
}

function setInstallHint(message, isError = false) {
  const node = document.getElementById("install-hint");
  if (!node) {
    return;
  }
  node.textContent = message || "";
  node.style.color = isError ? "#b42318" : "";
}

async function loadInstallStatus() {
  const status = await installRequest("/api/v1/install/status");
  document.title = installRuntime.app_title || "MVP IoT Install";

  const title = document.getElementById("install-title");
  if (title && status.app_name) {
    title.textContent = `${status.app_name} 安装向导`;
  }

  const pathNode = document.getElementById("setup-path");
  if (pathNode) {
    pathNode.textContent = status.setup_path || "managed by environment";
  }

  const pill = document.getElementById("install-status");
  if (pill) {
    pill.textContent = status.installed ? "已安装" : "等待安装";
  }

  if (status.installed) {
    window.location.replace("/");
    return false;
  }
  return true;
}

function bindInstallForm() {
  const form = document.getElementById("install-form");
  if (!form) {
    return;
  }

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      setInstallHint("正在保存安装配置...");
      await installRequest("/api/v1/install/bootstrap", {
        method: "POST",
        body: JSON.stringify({
          app_name: document.getElementById("app-name").value.trim(),
          site_url: document.getElementById("site-url").value.trim(),
          admin_username: document.getElementById("admin-username").value.trim(),
          admin_email: document.getElementById("admin-email").value.trim(),
          default_tenant_name: document.getElementById("tenant-name").value.trim(),
          default_tenant_slug: document.getElementById("tenant-slug").value.trim(),
        }),
      });
      setInstallHint("安装完成，正在进入后台...");
      window.setTimeout(() => {
        window.location.replace("/");
      }, 300);
    } catch (error) {
      setInstallHint(error.message, true);
    }
  });
}

async function bootstrapInstall() {
  const continueInstall = await loadInstallStatus();
  if (!continueInstall) {
    return;
  }
  bindInstallForm();
}

bootstrapInstall().catch((error) => {
  console.error(error);
  setInstallHint(error.message, true);
});
