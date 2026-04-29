package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tableRenderPostgres(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_postgres",
		Description: "A Render-managed Postgres database.",
		List: &plugin.ListConfig{
			Hydrate:    listRenderPostgres,
			KeyColumns: plugin.OptionalColumns([]string{"name", "owner_id", "environment_id", "suspended"}),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderPostgres,
			KeyColumns: plugin.SingleColumn("id"),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the database."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the database."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace that owns the database.", Transform: transform.FromField("Owner.Id")},
			{Name: "environment_id", Type: proto.ColumnType_STRING, Description: "The ID of the environment the database belongs to, if any."},
			{Name: "database_name", Type: proto.ColumnType_STRING, Description: "The Postgres database name."},
			{Name: "database_user", Type: proto.ColumnType_STRING, Description: "The default Postgres database user."},
			{Name: "version", Type: proto.ColumnType_STRING, Description: "The PostgreSQL major version."},
			{Name: "plan", Type: proto.ColumnType_STRING, Description: "The pricing/instance plan for the database."},
			{Name: "region", Type: proto.ColumnType_STRING, Description: "The region the database is hosted in."},
			{Name: "status", Type: proto.ColumnType_STRING, Description: "The runtime status of the database."},
			{Name: "role", Type: proto.ColumnType_STRING, Description: "The replica role of the database (primary or replica)."},
			{Name: "suspended", Type: proto.ColumnType_STRING, Description: "Whether the database is suspended (suspended / not_suspended)."},
			{Name: "suspenders", Type: proto.ColumnType_JSON, Description: "List of reasons the database has been suspended."},
			{Name: "high_availability_enabled", Type: proto.ColumnType_BOOL, Description: "Whether HA is enabled."},
			{Name: "disk_size_gb", Type: proto.ColumnType_INT, Description: "Allocated disk size in gigabytes."},
			{Name: "primary_postgres_id", Type: proto.ColumnType_STRING, Description: "ID of the primary database, when this row is a read replica.", Transform: transform.FromField("PrimaryPostgresID")},
			{Name: "read_replicas", Type: proto.ColumnType_JSON, Description: "Read replicas attached to this database."},
			{Name: "ip_allow_list", Type: proto.ColumnType_JSON, Description: "CIDR blocks permitted to connect to the database."},
			{Name: "owner", Type: proto.ColumnType_JSON, Description: "Summary of the workspace that owns the database."},
			{Name: "dashboard_url", Type: proto.ColumnType_STRING, Description: "URL to view the database in the Render Dashboard."},
			{Name: "expires_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the free-tier database will expire, if applicable."},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the database was created."},
			{Name: "updated_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the database was last updated."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderPostgres(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_postgres.listRenderPostgres", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListPostgresParams{Limit: &limit}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &client.NameParam{v}
	}
	if v := d.EqualsQualString("owner_id"); v != "" {
		params.OwnerId = &client.OwnerIdParam{v}
	}
	if v := d.EqualsQualString("environment_id"); v != "" {
		params.EnvironmentId = &client.EnvironmentIdParam{v}
	}
	if v := d.EqualsQualString("suspended"); v != "" {
		s := client.SuspendedParam{v}
		params.Suspended = &s
	}

	for {
		resp, err := c.ListPostgresWithResponse(ctx, params)
		if err != nil {
			logger.Error("render_postgres.listRenderPostgres", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list postgres failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, item.Postgres)
			lastCursor = item.Cursor
			if d.RowsRemaining(ctx) == 0 {
				return nil, nil
			}
		}

		if len(page) < limit {
			return nil, nil
		}
		params.Cursor = &lastCursor
	}
}

func getRenderPostgres(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	id := d.EqualsQualString("id")
	if id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_postgres.getRenderPostgres", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrievePostgresWithResponse(ctx, id)
	if err != nil {
		logger.Error("render_postgres.getRenderPostgres", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve postgres failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return *resp.JSON200, nil
}
