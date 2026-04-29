---
title: "Steampipe Table: render_blueprint - Query Render blueprints using SQL"
description: "Query Render blueprints (render.yaml IaC definitions)."
---

# Table: render_blueprint

A Render blueprint is a `render.yaml`-driven definition of one or more resources, sourced from a git repo. Blueprints can auto-sync to Render whenever the file changes on the configured branch.

The list endpoint returns summary fields only. Selecting `resources` triggers a per-row Retrieve call to fetch the resources defined by the blueprint.

## Examples

### Basic info

```sql+postgres
select id, name, repo, branch, status, last_sync, auto_sync
from   render_blueprint;
```

### Blueprints with errored sync

```sql+postgres
select id, name, repo, branch, last_sync
from   render_blueprint
where  status = 'error';
```

### Blueprints that haven't synced in a week

```sql+postgres
select id, name, repo, last_sync
from   render_blueprint
where  last_sync < now() - interval '7 days';
```

### Blueprints with auto-sync disabled

```sql+postgres
select id, name, repo, branch
from   render_blueprint
where  auto_sync = false;
```

### Resources defined by a specific blueprint

This selects `resources` and fires a per-row hydrate.

```sql+postgres
select b.name as blueprint, r ->> 'name' as resource_name, r ->> 'type' as resource_type
from   render_blueprint b,
       jsonb_array_elements(b.resources) r
where  b.id = 'bp-XXXX';
```
