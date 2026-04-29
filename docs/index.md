---
organization: Render
category: ["paas"]
display_name: "Render"
short_name: "render"
description: "Steampipe plugin to query services and other resources on Render."
engines: ["steampipe", "sqlite", "postgres", "export"]
---

# Render + Steampipe

[Render](https://render.com) is a unified cloud to build and run all your apps and websites.

[Steampipe](https://steampipe.io) is an open-source zero-ETL engine to instantly query cloud APIs using SQL.

List services in your Render workspace:

```sql
select
  name,
  type,
  suspended,
  created_at,
  service_details ->> 'region' as region
from
  render_service;
```

```
+-----------+-------------+---------------+---------------------------+-----------+
| name      | type        | suspended     | created_at                | region    |
+-----------+-------------+---------------+---------------------------+-----------+
| api       | web_service | not_suspended | 2024-08-12T17:42:01+00:00 | oregon    |
| worker    | background_worker | not_suspended | 2024-09-01T10:11:55+00:00 | oregon |
+-----------+-------------+---------------+---------------------------+-----------+
```

## Documentation

- **[Table definitions & examples →](/plugins/render-oss/render/tables)**

## Quick start

### Credentials

| Item        | Description                                                                                                                                       |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Credentials | All API requests require a Render [API key](https://render.com/docs/api) sent in the `Authorization` header as `Bearer <api_key>`.                 |
| Permissions | API keys carry the same privileges as the user that created them.                                                                                  |
| Radius      | Each connection represents a single Render workspace.                                                                                              |
| Resolution  | 1. Credentials in a Steampipe config file (`~/.steampipe/config/render.spc`)<br />2. The `RENDER_API_KEY` environment variable.                    |

### Configuration

```hcl
connection "render" {
  plugin = "render"

  # api_key = "rnd_AbCdEf1234567890"
  # api_url = "https://api.render.com/v1"  # optional override
}
```

Or via the environment:

```sh
export RENDER_API_KEY=rnd_AbCdEf1234567890
```
