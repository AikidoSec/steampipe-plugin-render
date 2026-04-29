package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tableRenderKeyValue(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_key_value",
		Description: "A Render Key Value (Redis-compatible) instance.",
		List: &plugin.ListConfig{
			Hydrate:    listRenderKeyValue,
			KeyColumns: plugin.OptionalColumns([]string{"name", "owner_id", "environment_id"}),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderKeyValue,
			KeyColumns: plugin.SingleColumn("id"),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the Key Value instance."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the instance."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace that owns the instance.", Transform: transform.FromField("Owner.Id")},
			{Name: "environment_id", Type: proto.ColumnType_STRING, Description: "The ID of the environment the instance belongs to, if any."},
			{Name: "version", Type: proto.ColumnType_STRING, Description: "The Key Value engine version (Redis or Valkey)."},
			{Name: "plan", Type: proto.ColumnType_STRING, Description: "The pricing/instance plan."},
			{Name: "region", Type: proto.ColumnType_STRING, Description: "The region the instance is hosted in."},
			{Name: "status", Type: proto.ColumnType_STRING, Description: "The runtime status of the instance."},
			{Name: "options", Type: proto.ColumnType_JSON, Description: "Engine options (e.g. maxmemoryPolicy)."},
			{Name: "ip_allow_list", Type: proto.ColumnType_JSON, Description: "CIDR blocks permitted to connect to the instance."},
			{Name: "owner", Type: proto.ColumnType_JSON, Description: "Summary of the workspace that owns the instance."},
			{Name: "dashboard_url", Type: proto.ColumnType_STRING, Description: "URL to view the instance in the Render Dashboard."},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the instance was created."},
			{Name: "updated_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the instance was last updated."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderKeyValue(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_key_value.listRenderKeyValue", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListKeyValueParams{Limit: &limit}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &client.NameParam{v}
	}
	if v := d.EqualsQualString("owner_id"); v != "" {
		params.OwnerId = &client.OwnerIdParam{v}
	}
	if v := d.EqualsQualString("environment_id"); v != "" {
		params.EnvironmentId = &client.EnvironmentIdParam{v}
	}

	for {
		resp, err := c.ListKeyValueWithResponse(ctx, params)
		if err != nil {
			logger.Error("render_key_value.listRenderKeyValue", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list key_value failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, item.KeyValue)
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

func getRenderKeyValue(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	id := d.EqualsQualString("id")
	if id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_key_value.getRenderKeyValue", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveKeyValueWithResponse(ctx, id)
	if err != nil {
		logger.Error("render_key_value.getRenderKeyValue", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve key_value failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return *resp.JSON200, nil
}
