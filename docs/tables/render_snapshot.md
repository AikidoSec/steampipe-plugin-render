---
title: "Steampipe Table: render_snapshot - Query Render disk snapshots using SQL"
description: "Query disk snapshots across Render disks."
---

# Table: render_snapshot

Point-in-time snapshots of Render disks.

The Render API only addresses snapshots per-disk. When you query without `disk_id`, this table walks every disk in every accessible workspace and lists snapshots for each — that's `1 + N + (N × M)` API calls (one ListOwners, N owners, then ListDisks per owner and ListSnapshots per disk). Pass `disk_id` whenever practical.

## Examples

### Snapshots for one disk

```sql+postgres
select snapshot_key, created_at, instance_id
from   render_snapshot
where  disk_id = 'dsk-XXXX'
order  by created_at desc;
```

### Most recent snapshot per disk

```sql+postgres
select   disk_id, max(created_at) as latest_snapshot
from     render_snapshot
group by disk_id
order by latest_snapshot;
```

### Disks with no snapshots in the last 7 days

```sql+postgres
select   d.id, d.name, d.service_id, max(s.created_at) as latest
from     render_disk     d
left join render_snapshot s on s.disk_id = d.id
group by d.id, d.name, d.service_id
having   max(s.created_at) is null
      or max(s.created_at) < now() - interval '7 days';
```
