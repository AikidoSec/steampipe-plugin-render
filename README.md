# Render plugin for Steampipe

Use SQL to query services and other resources from [Render](https://render.com).

> **Status:** initial scaffolding. Currently exposes only `render_service`.

## Quick start

### Install

```sh
make install
```

This builds the plugin into `~/.steampipe/plugins/hub.steampipe.io/plugins/render-oss/render@latest/`.

### Configure

Copy `config/render.spc` to `~/.steampipe/config/render.spc` and add your API key, or export it:

```sh
export RENDER_API_KEY=rnd_xxx
```

### Query

```sh
steampipe query "select name, type, suspended from render_service"
```

## Tables

| Table                          | Description                                                                                  |
| ------------------------------ | -------------------------------------------------------------------------------------------- |
| `render_blueprint`             | Blueprints (render.yaml IaC definitions). `resources` is hydrated lazily.                    |
| `render_custom_domain`         | Custom domains attached to web/static-site services.                                         |
| `render_deploy`                | Deploy history of a service. Provide `service_id` for fast queries.                          |
| `render_disk`                  | Persistent disks attached to services.                                                       |
| `render_env_group`             | Shared env var groups. Metadata only; secret-bearing env vars/files are not retrieved.       |
| `render_environment`           | Environments within a project (production, staging, ...).                                    |
| `render_header`                | Response-header rules on web/static-site services.                                           |
| `render_job`                   | One-off jobs run against a service.                                                          |
| `render_key_value`             | Key Value (Redis-compatible) instances.                                                      |
| `render_log_stream`            | Per-resource log-stream destination overrides (auth tokens not exposed).                     |
| `render_notification_override` | Per-service overrides of workspace notification settings.                                    |
| `render_owner`                 | Workspaces (users/teams). The `id` is what shows up as `owner_id` on other resources.        |
| `render_postgres`              | Render-managed Postgres databases (primaries and read replicas).                             |
| `render_postgres_export`       | Logical export jobs of Postgres databases (signed download URLs are deliberately omitted).    |
| `render_project`               | Projects, which group environments and their resources.                                      |
| `render_registry_credential`   | Container registry credentials.                                                              |
| `render_route`                 | Redirect/rewrite rules on web/static-site services.                                          |
| `render_secret_file`           | Names of secret files mounted into a service (contents not exposed).                         |
| `render_service`               | Services in a workspace (web, private, worker, cron, static site).                           |
| `render_snapshot`              | Point-in-time disk snapshots.                                                                |
| `render_webhook`               | Outbound webhook configurations (signing secret not exposed).                                |

## Regenerating the API client

The client under `render/client/` is generated from the [Render public API schema](https://github.com/renderinc/public-api-schema) using [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen).

```sh
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
PUBLIC_API_SCHEMA_PATH=/path/to/public-api-schema ./generate.sh
```

If `PUBLIC_API_SCHEMA_PATH` is unset, the script looks for `../public-api-schema` next to this repo.
