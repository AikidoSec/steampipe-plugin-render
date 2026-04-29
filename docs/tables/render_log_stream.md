---
title: "Steampipe Table: render_log_stream - Query Render log-stream destinations using SQL"
description: "Query per-resource log-stream overrides."
---

# Table: render_log_stream

Per-resource log-stream destination overrides. Each row tells you where logs from a specific resource are being shipped (or whether they're being dropped). The auth `token` used to authenticate to the destination is **not exposed** by the API list response.

Useful for data-exfiltration audits ("what external endpoints are receiving our logs?") and for checking that production resources are wired up to the workspace's central log destination.

## Examples

### All overrides

```sql+postgres
select resource_id, owner_id, setting, endpoint
from   render_log_stream;
```

### Distinct destination endpoints

```sql+postgres
select endpoint, count(*) as resources
from   render_log_stream
where  setting = 'send'
group  by endpoint
order  by resources desc;
```

### Resources where logs are being dropped

```sql+postgres
select s.name as service, ls.resource_id
from   render_log_stream ls
left join render_service s on s.id = ls.resource_id
where  ls.setting = 'drop';
```

### Endpoints not on the workspace's allow-list

Replace the IN clause with your approved destinations:

```sql+postgres
select resource_id, owner_id, endpoint
from   render_log_stream
where  setting  = 'send'
  and  endpoint not like '%logs.example.com/%'
  and  endpoint not like '%datadoghq.com/%';
```
