---
title: "Steampipe Table: render_notification_override - Query Render notification overrides using SQL"
description: "Query per-service overrides of workspace notification settings."
---

# Table: render_notification_override

Per-service overrides of the workspace-level notification setting. A row exists only when a service has overridden the workspace default; services using the default don't appear.

## Examples

### All overrides

```sql+postgres
select service_id, notifications_to_send, preview_notifications_enabled
from   render_notification_override;
```

### Services with notifications disabled

```sql+postgres
select s.name, no.notifications_to_send
from   render_notification_override no
join   render_service s on s.id = no.service_id
where  no.notifications_to_send = 'none';
```

### Override coverage by service type

```sql+postgres
select s.type, count(no.*) as overridden, count(distinct s.id) as total_services
from   render_service s
left   join render_notification_override no on no.service_id = s.id
group  by s.type;
```
