# Gitea-Inspired IoT Platform Architecture

This document maps the Gitea architecture to this IoT platform.

It is not a proposal to copy Gitea's Git business domain. It is a proposal to reuse the parts of Gitea that are structurally strong:

- startup lifecycle
- install lock
- config loading
- database and migration boundaries
- router/service/model separation
- admin-oriented backend layout

## What Gitea Gets Right

From the local `gitea/` clone in this workspace, the most useful references are:

- `gitea/main.go`
  - thin process entry
  - version/build metadata
  - delegates actual lifecycle to `cmd`
- `gitea/cmd/web.go`
  - single web command as the runtime entry
  - separates install mode and installed mode
  - graceful shutdown, startup logging, profiling hooks
- `gitea/modules/setting/setting.go`
  - config provider abstraction
  - `InstallLock` loaded very early
  - common settings loaded before runtime settings
- `gitea/routers/install/install.go`
  - dedicated install router
  - first-run form validation
  - config persistence and post-install bootstrap
- `gitea/models/db/engine.go`
  - shared database engine boundary
  - model registration and migration-oriented schema sync
- `gitea/models/*`, `gitea/services/*`, `gitea/routers/*`
  - domain model, service logic, router/controller split
- `gitea/web_src`, `gitea/templates`, `gitea/public`
  - frontend sources, templates, and built assets are clearly separated

## Current IoT Platform State

The current project is much smaller and still close to an MVP:

- `internal/api`
  - HTTP handlers plus embedded UI
- `internal/core`
  - main domain service
- `internal/store`
  - in-memory or file-backed persistence
- `internal/gateway`
  - TCP device access
- `internal/mqtt`
  - built-in MQTT broker
- `internal/simulator`
  - simulator management

This is workable for an MVP, but it becomes hard to evolve into a Gitea-like admin system because:

- install and runtime config were previously mixed with environment-only boot
- there was no install lock
- UI and API were too tightly bundled
- there is no dedicated system domain for instance metadata
- file store is fine for MVP, but not enough for active-active clustering

## What Has Already Been Added

The current codebase now has the first Gitea-like primitives:

- split executables
  - `mvp-backend`
  - `mvp-web-admin`
  - `mvp-desktop`
- install lock and first-run wizard
- install state replication for active/standby nodes
- runtime status endpoint and node topology endpoint
- standby mode and active/standby snapshot replication for the file store
- access logging, CORS, request IDs

That gives us a base similar to Gitea's "install first, then run as an admin backend" lifecycle.

## Recommended Target Layers

The target structure should move toward this shape:

```text
cmd/
  iotd/                 runtime process entry
  iot-web-admin/        standalone admin UI
  iot-desktop/          desktop shell

internal/
  bootstrap/            install lock, setup wizard, instance bootstrap
  config/               env + file config provider
  system/               instance settings, admin users, audit, jobs
  repository/           database repositories
  service/              business services
  web/
    admin/              admin HTTP routes
    api/                external API routes
    install/            install routes
    middleware/         auth, install guard, logging, tracing
  transport/
    tcp/
    mqtt/
    httpingest/
  domain/
    tenant/
    product/
    device/
    group/
    rule/
    alert/
    configprofile/
    firmware/
    ota/
  frontend/
    admin/
    install/
```

## Direct Mapping From Gitea To This IoT Platform

Gitea concept -> IoT equivalent:

- `InstallLock` -> platform installation state
- `app.ini` -> instance config file plus environment overrides
- `models/db` -> repository and migration layer
- `routers/install` -> `/api/v1/install/*` plus install UI
- `routers/web` -> admin console routes
- `services/*` -> product, device, rule, OTA, simulator, and governance services
- `admin dashboard` -> platform operation center
- `system notice / cron / queue / config` -> ingestion jobs, OTA tasks, command queue, alarm processing, retention jobs

## Backend Style The Admin UI Should Follow

If the visual goal is "Gitea-like admin backend", the UI should emphasize:

- left navigation with dense functional grouping
- top-level operation dashboard
- system administration pages
- tables first, detail pages second
- install page before first login
- configuration pages that look operational, not consumer-facing

For this IoT platform, the primary admin sections should be:

- Overview
  - runtime, health, node role, storage, ingestion volume
- Tenant Workspace
  - tenant list, quotas, isolation, usage
- Product Center
  - thing models, access profiles, protocol templates
- Device Center
  - device registry, status, commands, shadows, telemetry
- Governance
  - groups, rules, alerts, acknowledgements
- Config And OTA
  - config profiles, firmware artifacts, OTA campaigns
- Access And Transport
  - TCP gateway, MQTT broker, HTTP ingest, edge adapters
- Operations
  - audit log, job history, replication state, persistence state
- System Admin
  - install state, node config, backup, retention, integrations

## First-Open Install Flow

The install flow should mirror Gitea's behavior:

1. Start backend.
2. If no install state exists, normal business APIs are blocked.
3. The first opened UI is redirected to `/install`.
4. The operator saves:
   - platform name
   - public site URL
   - initial admin metadata
   - default tenant seed
5. Backend writes the install state file.
6. Backend unlocks normal APIs.
7. UI redirects to the admin dashboard.

In the current implementation, the install state is also replicated to standby nodes, so a backup node does not reopen the install wizard after the primary has already been initialized.

Later iterations should expand this install step to support:

- database selection
- external object storage
- initial RBAC admin creation
- mail / webhook / SMS / DingTalk / WeCom settings
- HA topology selection

## Data Model Direction

To keep the data density and admin usability close to Gitea, the system should expose summary views, not only raw entities.

Examples:

- TenantView
  - counts for products, devices, groups, rules, firmware, OTA
- ProductView
  - device count, online count, last ingest, protocol type
- DeviceView
  - tenant, product, groups, online state, last telemetry, last command result
- RuleView
  - trigger count, last triggered at, action summary
- OTACampaignView
  - target scope, dispatched count, acked count, failed count
- SystemView
  - install state, store backend, replication state, queue depth, retention status

This "view model" direction is closer to how Gitea's admin pages present aggregate operational data.

## What Not To Copy From Gitea

Do not copy these parts directly:

- Git-specific domain layout
- repository-centric routing names
- template naming conventions tied to code hosting
- user/org permission semantics without adapting to tenant/device/RBAC semantics

The architecture should be inspired by Gitea, but the domain must remain IoT-native.

## Next Refactor Steps

Recommended order:

1. Keep the current install lock and split executables.
2. Introduce a real persistent database backend and migrations.
3. Separate `store` into repository interfaces plus database implementations.
4. Split current `internal/core` into domain services.
5. Add system admin domain: users, roles, audit logs, jobs, instance settings.
6. Rebuild the admin UI into Gitea-like dense operations pages.
7. Add login and RBAC after install bootstrap is stable.
