---
title: "Steampipe Table: render_disk - Query Render disks using SQL"
description: "Query persistent disks attached to Render services."
---

# Table: render_disk

A persistent disk attached to a Render service.

## Examples

### Basic info

```sql+postgres
select id, name, mount_path, size_gb, service_id
from   render_disk;
```

### Total provisioned disk per service

```sql+postgres
select s.name as service, sum(d.size_gb) as gb
from   render_disk    d
join   render_service s on s.id = d.service_id
group  by s.name
order  by gb desc;
```

### Largest disks

```sql+postgres
select name, mount_path, size_gb, service_id
from   render_disk
order  by size_gb desc
limit 10;
```

### Unattached disks (rare; may indicate cleanup needed)

```sql+postgres
select id, name, size_gb, created_at
from   render_disk
where  service_id is null;
```
