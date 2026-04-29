---
title: "Steampipe Table: render_postgres_export - Query Render Postgres exports using SQL"
description: "Query logical export jobs of Render Postgres databases."
---

# Table: render_postgres_export

Logical exports (dumps) of Render-managed Postgres databases. Each row is one export job. The `url` column is a temporary signed download URL — sensitive while valid; treat it like a short-lived credential.

Pass `postgres_id` when you can; without it, this table walks every Postgres database in the workspace.

## Examples

### Recent exports for one database

```sql+postgres
select id, created_at, url is not null as has_url
from   render_postgres_export
where  postgres_id = 'dpg-XXXX'
order  by created_at desc;
```

### Cross-workspace export activity in the last 30 days

```sql+postgres
select pe.created_at, p.name as database, p.owner_id, pe.id as export_id
from   render_postgres_export pe
join   render_postgres        p on p.id = pe.postgres_id
where  pe.created_at > now() - interval '30 days'
order  by pe.created_at desc;
```

### Databases whose latest export is stale

```sql+postgres
select   p.name, p.id, max(pe.created_at) as last_export
from     render_postgres p
left join render_postgres_export pe on pe.postgres_id = p.id
group by p.name, p.id
having   max(pe.created_at) is null
      or max(pe.created_at) < now() - interval '30 days';
```
