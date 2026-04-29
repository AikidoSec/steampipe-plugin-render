---
title: "Steampipe Table: render_registry_credential - Query Render registry credentials using SQL"
description: "Query container-registry credentials in a Render workspace."
---

# Table: render_registry_credential

A registry credential lets Render pull private container images for image-deployed services. Each credential targets a specific registry (Docker Hub, GHCR, GAR, etc.) and is referenced by name on services that need it.

## Examples

### Basic info

```sql+postgres
select id, name, registry, username, updated_at
from   render_registry_credential;
```

### Credentials by registry type

```sql+postgres
select registry, count(*)
from   render_registry_credential
group  by registry
order  by count(*) desc;
```

### Stale credentials (not rotated in 90 days)

```sql+postgres
select id, name, registry, username, updated_at
from   render_registry_credential
where  updated_at < now() - interval '90 days'
order  by updated_at;
```

### Credentials referenced by services

```sql+postgres
select c.name as credential, c.registry, count(s.*) as services_using
from   render_registry_credential c
left join render_service s on (s.registry_credential ->> 'id') = c.id
group  by c.name, c.registry
order  by services_using desc;
```
