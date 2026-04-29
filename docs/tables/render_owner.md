---
title: "Steampipe Table: render_owner - Query Render workspaces using SQL"
description: "Query Render workspace owners (users or teams)."
---

# Table: render_owner

A Render `owner` is a user or team workspace. The owner ID (`tea-...` or `usr-...`) is the workspace identifier used as `owner_id` on services, projects, and other resources.

## Examples

### Basic info

```sql+postgres
select
  id,
  name,
  email,
  type
from
  render_owner;
```

### Find a specific workspace by name

```sql+postgres
select id, name, email
from   render_owner
where  name = 'Render';
```

### Users without 2FA

```sql+postgres
select id, name, email
from   render_owner
where  type = 'user'
  and  two_factor_auth_enabled is not true;
```
