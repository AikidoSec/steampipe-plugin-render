---
title: "Steampipe Table: render_env_group - Query Render env groups using SQL"
description: "Query Render env groups and the services they're linked to."
---

# Table: render_env_group

An env group is a shared set of environment variables (and optionally secret files) that can be linked to multiple services.

This table intentionally exposes metadata only. It does not retrieve `env_vars` or `secret_files`, because the full env-group payload can contain secret values and file contents.

## Examples

### Basic info

```sql+postgres
select id, name, owner_id, environment_id
from   render_env_group;
```

### Env groups and the services they're linked to

```sql+postgres
select g.name as env_group, link ->> 'name' as service, link ->> 'type' as service_type
from   render_env_group g,
       jsonb_array_elements(g.service_links) link;
```

### Env groups linked to multiple services

```sql+postgres
select name, jsonb_array_length(service_links) as linked_services
from   render_env_group
where  jsonb_array_length(service_links) > 1;
```

### Env groups with no linked services

```sql+postgres
select id, name
from   render_env_group
where  jsonb_array_length(service_links) = 0;
```
