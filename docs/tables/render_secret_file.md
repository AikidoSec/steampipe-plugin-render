---
title: "Steampipe Table: render_secret_file - Query Render secret-file mounts using SQL"
description: "List names of secret files mounted into Render services."
---

# Table: render_secret_file

Names of secret files mounted into a Render service. **Contents are deliberately not exposed** by this table even though the API returns them — use the Render API directly if you need to read a secret file's contents.

## Examples

### All secret files in the workspace

```sql+postgres
select name, service_id from render_secret_file;
```

### Secret files for one service

```sql+postgres
select name from render_secret_file where service_id = 'srv-XXXX';
```

### Services using a particular secret file name

```sql+postgres
select service_id from render_secret_file where name = '.env.production';
```
