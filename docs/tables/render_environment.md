---
title: "Steampipe Table: render_environment - Query Render environments using SQL"
description: "Query environments inside Render projects."
---

# Table: render_environment

An environment groups resources within a project (e.g. `production`, `staging`). Environments can be marked `protected`, in which case only admins may perform destructive actions.

Listing without a `project_id` qualifier walks every project in the workspace.

## Examples

### Basic info

```sql+postgres
select id, name, project_id, protected_status
from   render_environment;
```

### Environments inside a specific project

```sql+postgres
select id, name, protected_status
from   render_environment
where  project_id = 'prj-abc123';
```

### Production environments missing network isolation

```sql+postgres
select e.name as environment, p.name as project
from   render_environment e
join   render_project     p on p.id = e.project_id
where  e.name = 'production'
  and  e.network_isolation_enabled is not true;
```

### Resource counts per environment

```sql+postgres
select
  name,
  jsonb_array_length(service_ids)   as service_count,
  jsonb_array_length(databases_ids) as postgres_count,
  jsonb_array_length(redis_ids)     as redis_count
from
  render_environment
order by service_count desc;
```
