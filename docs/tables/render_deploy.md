---
title: "Steampipe Table: render_deploy - Query Render service deploys using SQL"
description: "Query the deploy history of Render services."
---

# Table: render_deploy

A Render deploy is a build + release of a service. Each deploy has a status (`live`, `build_failed`, `canceled`, etc.), a trigger (`new_commit`, `manual_deploy`, ...), and either commit metadata (Git-backed services) or image metadata (image-deployed services).

Listing without a `service_id` qualifier walks every service in the workspace, which can be slow on large accounts. Provide `service_id` whenever you can.

## Examples

### Most recent deploys for one service

```sql+postgres
select id, status, trigger, commit_message, created_at, finished_at
from   render_deploy
where  service_id = 'srv-abc123'
order by created_at desc
limit 10;
```

### Failed deploys in the last 7 days, across the workspace

```sql+postgres
select s.name as service, d.id as deploy_id, d.status, d.commit_message, d.created_at
from   render_deploy  d
join   render_service s on s.id = d.service_id
where  d.status in ('build_failed', 'update_failed', 'pre_deploy_failed')
  and  d.created_at > now() - interval '7 days'
order by d.created_at desc;
```

### Average build time per service (last 50 successful deploys each)

```sql+postgres
with recent as (
  select service_id, finished_at - created_at as duration,
         row_number() over (partition by service_id order by created_at desc) as rn
  from   render_deploy
  where  status = 'live' and finished_at is not null
)
select s.name, avg(r.duration) as avg_deploy_time
from   recent r
join   render_service s on s.id = r.service_id
where  r.rn <= 50
group  by s.name
order  by avg_deploy_time desc;
```

### Deploys triggered by a specific commit SHA

```sql+postgres
select service_id, id, status, created_at
from   render_deploy
where  commit_id = 'a1b2c3d4e5f6';
```
