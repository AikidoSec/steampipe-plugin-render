---
title: "Steampipe Table: render_job - Query Render one-off jobs using SQL"
description: "Query one-off jobs run against Render services."
---

# Table: render_job

A one-off job (background script, migration, etc.) launched against a Render service. Walking the table without `service_id` iterates every service in the workspace.

## Examples

### Recent jobs for one service

```sql+postgres
select id, status, start_command, created_at, finished_at
from   render_job
where  service_id = 'srv-XXXX'
order  by created_at desc
limit 20;
```

### Failed jobs in the last 7 days, across the workspace

```sql+postgres
select s.name as service, j.id, j.status, j.start_command, j.created_at
from   render_job     j
join   render_service s on s.id = j.service_id
where  j.status = 'failed'
  and  j.created_at > now() - interval '7 days'
order  by j.created_at desc;
```

### Average job runtime per service

```sql+postgres
select s.name, avg(j.finished_at - j.started_at) as avg_runtime
from   render_job     j
join   render_service s on s.id = j.service_id
where  j.status = 'succeeded'
  and  j.finished_at is not null
group  by s.name
order  by avg_runtime desc;
```
