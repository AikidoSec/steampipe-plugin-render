package render

import (
	"context"
	"fmt"
	"net/http"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

type Header struct {
	client.Header
	ServiceId string
}

func tableRenderHeader(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_header",
		Description: "HTTP response header rules attached to Render static sites. (Web services don't support header-rule listing via the API.)",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderServices,
			Hydrate:       listRenderHeaders,
			KeyColumns: plugin.KeyColumnSlice{
				{Name: "service_id", Require: plugin.Optional},
				{Name: "name", Require: plugin.Optional},
				{Name: "path", Require: plugin.Optional},
			},
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the header rule."},
			{Name: "service_id", Type: proto.ColumnType_STRING, Description: "The ID of the service the rule is attached to."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The header name."},
			{Name: "value", Type: proto.ColumnType_STRING, Description: "The header value."},
			{Name: "path", Type: proto.ColumnType_STRING, Description: "The request path the header applies to (supports wildcards)."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderHeaders(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	service := h.Item.(client.Service)

	// Only static sites support header rules in the Render API. Asking for
	// headers on any other service type returns 400, which would kill the
	// whole query when walking all services.
	if service.Type != client.StaticSite {
		return nil, nil
	}

	if v := d.EqualsQualString("service_id"); v != "" && v != service.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_header.listRenderHeaders", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListHeadersParams{Limit: &limit}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &[]string{v}
	}
	if v := d.EqualsQualString("path"); v != "" {
		params.Path = &[]string{v}
	}

	for {
		resp, err := callWithRetry(ctx, func() (*client.ListHeadersResponse, *http.Response, error) {
			r, e := c.ListHeadersWithResponse(ctx, service.Id, params)
			if r != nil {
				return r, r.HTTPResponse, e
			}
			return r, nil, e
		})
		if err != nil {
			logger.Error("render_header.listRenderHeaders", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list headers failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor string
		for _, item := range page {
			d.StreamListItem(ctx, Header{Header: item.Header, ServiceId: service.Id})
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
