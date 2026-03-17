# Frontend And Backend Split Guide

This repository now supports three standalone programs:

- `mvp-backend`: backend service with HTTP API, device gateway, MQTT broker, access logging, CORS, health probes, and active/standby deployment mode.
- `mvp-web-admin`: standalone web admin console that serves the UI and talks to the backend over HTTP.
- `mvp-desktop`: desktop launcher that serves the same UI locally and opens it in the system browser.

## Build

```powershell
go build -o bin\mvp-backend.exe .\cmd\mvp-backend
go build -o bin\mvp-web-admin.exe .\cmd\mvp-web-admin
go build -o bin\mvp-desktop.exe .\cmd\mvp-desktop
```

## Backend

Primary backend:

```powershell
$env:MVP_NODE_ID="backend-a"
$env:MVP_NODE_ROLE="primary"
$env:MVP_DISABLE_EMBEDDED_UI="true"
$env:MVP_CORS_ALLOWED_ORIGINS="http://127.0.0.1:8081,http://127.0.0.1:18081"
.\bin\mvp-backend.exe
```

Standby backend:

```powershell
$env:MVP_NODE_ID="backend-b"
$env:MVP_NODE_ROLE="standby"
$env:MVP_DISABLE_EMBEDDED_UI="true"
$env:MVP_HA_REPLICA_TOKEN="replace-with-shared-secret"
.\bin\mvp-backend.exe
```

Primary backend with snapshot replication to the standby node:

```powershell
$env:MVP_NODE_ID="backend-a"
$env:MVP_NODE_ROLE="primary"
$env:MVP_DISABLE_EMBEDDED_UI="true"
$env:MVP_HA_REPLICA_TOKEN="replace-with-shared-secret"
$env:MVP_HA_REPLICA_PEERS="http://10.0.0.12:8080"
.\bin\mvp-backend.exe
```

Relevant backend environment variables:

- `MVP_NODE_ID`: logical node identifier for logs and topology endpoints.
- `MVP_NODE_ROLE`: `primary` or `standby`.
- `MVP_DISABLE_EMBEDDED_UI`: disables the legacy embedded console for split deployments.
- `MVP_CORS_ALLOWED_ORIGINS`: comma-separated allowed origins for the web admin and desktop UI.
- `MVP_HA_REPLICA_TOKEN`: shared secret for the standby snapshot endpoint.
- `MVP_HA_REPLICA_PEERS`: comma-separated backend base URLs that receive `/_ha/snapshot`.
- `MVP_HA_REPLICA_TIMEOUT`: timeout for snapshot pushes.

Operational endpoints:

- `/healthz`: process liveness.
- `/readyz`: returns `200` only on the primary node. Standby nodes return `503`.
- `/metrics`: JSON or Prometheus metrics.
- `/api/v1/system/info`: node role, UI mode, store backend, and topology details.

## Web Admin

```powershell
$env:MVP_WEB_ADDR=":8081"
$env:MVP_API_BASE_URL="http://127.0.0.1:8080"
.\bin\mvp-web-admin.exe
```

## Desktop

```powershell
$env:MVP_DESKTOP_ADDR="127.0.0.1:18081"
$env:MVP_API_BASE_URL="http://127.0.0.1:8080"
.\bin\mvp-desktop.exe
```

Set `MVP_DESKTOP_OPEN_BROWSER=false` if the desktop launcher should not open the browser automatically.

## HA And Load Balancing Notes

The current file-backed store now supports active/standby replication by pushing persisted snapshots from the primary node to one or more standby nodes.

This is suitable for:

- dual-machine hot backup
- failover behind a load balancer using `/readyz`
- centralized structured logging from both nodes

This is not true active-active write balancing. If both backend nodes accept writes while using the local file store, state will diverge.

For real active-active load balancing, replace the local file store with a shared external datastore and run both nodes as `primary`.
