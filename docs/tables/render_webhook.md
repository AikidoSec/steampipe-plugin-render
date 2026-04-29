---
title: "Steampipe Table: render_webhook - Query Render outbound webhooks using SQL"
description: "Query outbound webhook configurations across Render workspaces."
---

# Table: render_webhook

Outbound webhooks Render fires when events happen in a workspace. The HMAC signing `secret` returned by the API is **deliberately not exposed** as a column — read it from the API directly if you need it.

## Examples

### Basic info

```sql+postgres
select id, name, url, enabled, owner_id from render_webhook;
```

### Webhooks pointing to suspicious destinations

```sql+postgres
select w.name, w.url, o.name as workspace, o.email
from   render_webhook w
join   render_owner   o on o.id = w.owner_id
where  w.url not like 'https://%'
   or  w.url ~ '\.(ru|cn|tk|ml)/';
```

### Disabled webhooks (potential dead config)

```sql+postgres
select id, name, url, owner_id
from   render_webhook
where  enabled = false;
```

### Webhooks subscribed to ALL events (broad scope)

```sql+postgres
select id, name, url, owner_id
from   render_webhook
where  jsonb_array_length(event_filter) = 0;
```

### Webhooks subscribed to deploy events only

```sql+postgres
select id, name, url
from   render_webhook
where  event_filter @> '["deploy_started"]'
   or  event_filter @> '["deploy_succeeded"]'
   or  event_filter @> '["deploy_failed"]';
```
