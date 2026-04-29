---
title: "Steampipe Table: render_env_group - Query Render env groups using SQL"
description: "Query Render env groups and the services they're linked to."
---

# Table: render_env_group

An env group is a shared set of environment variables (and optionally secret files) that can be linked to multiple services.

By default, listing returns metadata only (no `env_vars` or `secret_files`). Selecting either of those columns triggers a per-row API call to fetch the full group, so use them deliberately when querying many groups.

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

### Get the keys (not values) inside one env group

Selecting `env_vars` triggers a hydrate that fetches the full record. Secret values come back as nulls.

```sql+postgres
select name, jsonb_agg(v ->> 'key') as keys
from   render_env_group g,
       jsonb_array_elements(g.env_vars) v
where  g.id = 'evg-XXXX'
group  by name;
```

### Env groups with no linked services

```sql+postgres
select id, name
from   render_env_group
where  jsonb_array_length(service_links) = 0;
```
