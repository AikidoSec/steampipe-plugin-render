---
title: "Steampipe Table: render_key_value - Query Render Key Value instances using SQL"
description: "Query Render Key Value (Redis-compatible) instances."
---

# Table: render_key_value

A Render Key Value instance (Redis or Valkey, depending on `version`). This is Render's modern Redis-compatible offering; legacy `render_redis` instances are not covered here.

## Examples

### Basic info

```sql+postgres
select id, name, region, plan, version, status
from   render_key_value;
```

### Eviction policy summary

```sql+postgres
select name, options ->> 'maxmemoryPolicy' as eviction_policy, plan
from   render_key_value
order  by eviction_policy nulls last;
```

### Instances with public IP allow rules

```sql+postgres
select name, ip_allow_list
from   render_key_value
where  ip_allow_list @> '[{"cidrBlock": "0.0.0.0/0"}]';
```

### Instances by environment

```sql+postgres
select e.name as environment, kv.name as instance, kv.plan
from   render_key_value kv
join   render_environment e on e.id = kv.environment_id
order  by e.name, kv.name;
```
