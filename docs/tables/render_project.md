---
title: "Steampipe Table: render_project - Query Render projects using SQL"
description: "Query Render projects, which group environments and the resources within them."
---

# Table: render_project

A Render project is a top-level container for environments and the services, databases, and key-value stores belonging to them.

## Examples

### Basic info

```sql+postgres
select
  id,
  name,
  owner_id,
  created_at
from
  render_project;
```

### Projects in a specific workspace

```sql+postgres
select id, name, jsonb_array_length(environment_ids) as env_count
from   render_project
where  owner_id = 'tea-abc123';
```

### Project + owner email (join)

```sql+postgres
select p.name as project_name, o.email
from   render_project p
join   render_owner   o on o.id = p.owner_id;
```
