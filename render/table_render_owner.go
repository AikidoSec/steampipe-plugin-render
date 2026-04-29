package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tableRenderOwner(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_owner",
		Description: "A Render workspace owner. The Render API uses 'owner' to refer to a user or team workspace.",
		List: &plugin.ListConfig{
			Hydrate:    listRenderOwners,
			KeyColumns: plugin.OptionalColumns([]string{"name", "email"}),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderOwner,
			KeyColumns: plugin.SingleColumn("id"),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the owner (workspace)."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the owner."},
			{Name: "email", Type: proto.ColumnType_STRING, Description: "The email address associated with the owner."},
			{Name: "type", Type: proto.ColumnType_STRING, Description: "The owner type (user or team)."},
			{Name: "two_factor_auth_enabled", Type: proto.ColumnType_BOOL, Description: "Whether two-factor authentication is enabled. Only set when type is 'user'."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderOwners(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_owner.listRenderOwners", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListOwnersParams{Limit: &limit}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &[]string{v}
	}
	if v := d.EqualsQualString("email"); v != "" {
		params.Email = &[]string{v}
	}

	for {
		resp, err := c.ListOwnersWithResponse(ctx, params)
		if err != nil {
			logger.Error("render_owner.listRenderOwners", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list owners failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			if item.Owner == nil {
				continue
			}
			d.StreamListItem(ctx, *item.Owner)
			if item.Cursor != nil {
				lastCursor = *item.Cursor
			}
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

func getRenderOwner(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	id := d.EqualsQualString("id")
	if id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_owner.getRenderOwner", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveOwnerWithResponse(ctx, id)
	if err != nil {
		logger.Error("render_owner.getRenderOwner", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve owner failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return *resp.JSON200, nil
}
