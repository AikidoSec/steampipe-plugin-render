package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

type Route struct {
	client.Route
	ServiceId string
}

func tableRenderRoute(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_route",
		Description: "Redirect and rewrite rules attached to Render web or static-site services.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderServices,
			Hydrate:       listRenderRoutes,
			KeyColumns: plugin.KeyColumnSlice{
				{Name: "service_id", Require: plugin.Optional},
				{Name: "type", Require: plugin.Optional},
				{Name: "source", Require: plugin.Optional},
				{Name: "destination", Require: plugin.Optional},
			},
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the route."},
			{Name: "service_id", Type: proto.ColumnType_STRING, Description: "The ID of the service the route is attached to."},
			{Name: "type", Type: proto.ColumnType_STRING, Description: "The route type (redirect or rewrite)."},
			{Name: "source", Type: proto.ColumnType_STRING, Description: "The source path pattern."},
			{Name: "destination", Type: proto.ColumnType_STRING, Description: "The destination path."},
			{Name: "priority", Type: proto.ColumnType_INT, Description: "Priority order; routes are evaluated starting at priority 0."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Source")},
		},
	}
}

func listRenderRoutes(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	service := h.Item.(client.Service)

	if v := d.EqualsQualString("service_id"); v != "" && v != service.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_route.listRenderRoutes", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListRoutesParams{Limit: &limit}
	if v := d.EqualsQualString("type"); v != "" {
		t := []client.ListRoutesParamsType{client.ListRoutesParamsType(v)}
		params.Type = &t
	}
	if v := d.EqualsQualString("source"); v != "" {
		params.Source = &[]string{v}
	}
	if v := d.EqualsQualString("destination"); v != "" {
		params.Destination = &[]string{v}
	}

	for {
		resp, err := c.ListRoutesWithResponse(ctx, service.Id, params)
		if err != nil {
			logger.Error("render_route.listRenderRoutes", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list routes failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor string
		for _, item := range page {
			d.StreamListItem(ctx, Route{Route: item.Route, ServiceId: service.Id})
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
