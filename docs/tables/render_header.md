---
title: "Steampipe Table: render_header - Query Render header rules using SQL"
description: "Query response-header rules on Render web and static-site services."
---

# Table: render_header

Response-header rules attached to a Render web or static-site service. Useful for auditing security headers (CSP, HSTS, X-Frame-Options) across the workspace.

## Examples

### Every header rule across the workspace

```sql+postgres
select service_id, name, value, path from render_header;
```

### Services missing common security headers

```sql+postgres
with s as (select id, name from render_service where type in ('web_service','static_site'))
select s.name as service
from   s
where  not exists (
  select 1 from render_header h
  where h.service_id = s.id and h.name ilike 'strict-transport-security'
);
```

### CSP coverage

```sql+postgres
select h.service_id, h.path, h.value
from   render_header h
where  h.name ilike 'content-security-policy';
```
