---
title: "Steampipe Table: render_service - Query Render Services using SQL"
description: "Query Render services (web services, private services, background workers, cron jobs, static sites)."
---

# Table: render_service

Render services are the runnable units of work in a workspace: web services, private services, background workers, cron jobs, and static sites.

## Examples

### Basic info

```sql+postgres
select
  name,
  id,
  type,
  suspended,
  created_at
from
  render_service;
```

### Services by region and plan

```sql+postgres
select
  name,
  type,
  service_details ->> 'region' as region,
  service_details ->> 'plan'   as plan
from
  render_service
order by
  region, plan;
```

### Suspended services

```sql+postgres
select
  name,
  type,
  suspenders,
  dashboard_url
from
  render_service
where
  suspended = 'suspended';
```

### Services deploying from a specific repo

```sql+postgres
select
  name,
  branch,
  auto_deploy,
  type
from
  render_service
where
  repo = 'https://github.com/render-oss/render-mcp-server';
```

### Filter at the API: list only web services for a workspace

The `name`, `type`, `environment_id`, `owner_id`, and `suspended` qualifiers are pushed down to the Render API.

```sql+postgres
select
  name,
  service_details ->> 'region' as region,
  service_details ->> 'plan'   as plan
from
  render_service
where
  type     = 'web_service'
  and owner_id = 'tea-abc123';
```
