---
title: "Steampipe Table: render_route - Query Render redirect/rewrite rules using SQL"
description: "Query redirect and rewrite rules on Render web and static-site services."
---

# Table: render_route

Redirect and rewrite rules attached to a Render web or static-site service. Routes are evaluated in `priority` order starting at 0.

## Examples

### Routes for one service, in evaluation order

```sql+postgres
select priority, type, source, destination
from   render_route
where  service_id = 'srv-XXXX'
order  by priority;
```

### All redirects across the workspace

```sql+postgres
select service_id, source, destination
from   render_route
where  type = 'redirect';
```

### Services with the most route rules

```sql+postgres
select s.name as service, count(r.*) as route_count
from   render_service s
join   render_route   r on r.service_id = s.id
group  by s.name
order  by route_count desc
limit 10;
```
