---
title: "Steampipe Table: render_postgres - Query Render Postgres databases using SQL"
description: "Query Render-managed Postgres databases."
---

# Table: render_postgres

A Render-managed Postgres database. Includes both primaries and read replicas (filter by `role` to separate them).

## Examples

### Basic info

```sql+postgres
select id, name, region, plan, version, status
from   render_postgres;
```

### Databases without high availability

```sql+postgres
select name, region, plan, version
from   render_postgres
where  high_availability_enabled is not true
  and  role = 'primary';
```

### Free-tier databases approaching expiry

```sql+postgres
select name, expires_at, owner_id
from   render_postgres
where  expires_at is not null
  and  expires_at < now() + interval '14 days'
order by expires_at;
```

### Databases with permissive IP allow lists (open to the world)

```sql+postgres
select name, ip_allow_list
from   render_postgres
where  ip_allow_list @> '[{"cidrBlock": "0.0.0.0/0"}]';
```

### Read-replica topology

```sql+postgres
select primary_postgres_id, count(*) as replica_count
from   render_postgres
where  role = 'replica'
group  by primary_postgres_id;
```
