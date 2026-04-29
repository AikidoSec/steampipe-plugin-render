---
title: "Steampipe Table: render_custom_domain - Query Render custom domains using SQL"
description: "Query custom domains attached to Render web and static-site services."
---

# Table: render_custom_domain

A custom domain attached to a Render web or static-site service. Walking the table without `service_id` iterates every service in the workspace; pass `service_id` whenever possible.

## Examples

### All custom domains, with service IDs

```sql+postgres
select id, name, service_id, verification_status, domain_type
from   render_custom_domain;
```

### Unverified domains

```sql+postgres
select name, service_id, created_at
from   render_custom_domain
where  verification_status = 'unverified'
order  by created_at;
```

### Domains for one service

```sql+postgres
select name, domain_type, verification_status
from   render_custom_domain
where  service_id = 'srv-XXXX';
```

### Domain count per service

```sql+postgres
select s.name as service, count(d.*) as domains
from   render_service       s
left join render_custom_domain d on d.service_id = s.id
group  by s.name
order  by domains desc;
```
