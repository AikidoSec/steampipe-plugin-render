package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/render-oss/steampipe-plugin-render/render/client/logs"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// LogStream is a per-resource log-stream override. Note: the auth token used to
// authenticate with the destination is intentionally not in the list response.
type LogStream struct {
	ResourceId string
	OwnerId    string
	Setting    string
	Endpoint   string
}

func tableRenderLogStream(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_log_stream",
		Description: "Per-resource log-stream destination overrides. Authentication tokens are not exposed.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderOwners,
			Hydrate:       listRenderLogStreams,
			KeyColumns:    plugin.OptionalColumns([]string{"owner_id", "resource_id", "setting"}),
		},
		Columns: []*plugin.Column{
			{Name: "resource_id", Type: proto.ColumnType_STRING, Description: "The ID of the resource the override applies to (server, cron job, postgres, or redis)."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace this log stream belongs to."},
			{Name: "setting", Type: proto.ColumnType_STRING, Description: "Whether logs are sent to the destination or dropped."},
			{Name: "endpoint", Type: proto.ColumnType_STRING, Description: "The destination URL logs are streamed to (only set when setting=send)."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("ResourceId")},
		},
	}
}

func listRenderLogStreams(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	owner := h.Item.(client.Owner)

	if v := d.EqualsQualString("owner_id"); v != "" && v != owner.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_log_stream.listRenderLogStreams", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}
	params := &client.ListResourceLogStreamsParams{
		Limit:   &limit,
		OwnerId: &client.OwnerIdParam{owner.Id},
	}
	if v := d.EqualsQualString("resource_id"); v != "" {
		params.ResourceId = &client.ResourceIdParam{v}
	}
	if v := d.EqualsQualString("setting"); v != "" {
		f := logs.LogStreamSettingFilter{logs.LogStreamSetting(v)}
		params.Setting = &f
	}

	resp, err := c.ListResourceLogStreamsWithResponse(ctx, params)
	if err != nil {
		logger.Error("render_log_stream.listRenderLogStreams", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list log_streams failed: %s: %s", resp.Status(), string(resp.Body))
	}

	// The list endpoint returns []ResourceLogStreamSetting with no cursor, so a
	// single page is all we get per owner.
	for _, item := range *resp.JSON200 {
		row := LogStream{OwnerId: owner.Id}
		if item.ResourceId != nil {
			row.ResourceId = *item.ResourceId
		}
		if item.Setting != nil {
			row.Setting = string(*item.Setting)
		}
		if item.Endpoint != nil {
			row.Endpoint = *item.Endpoint
		}
		d.StreamListItem(ctx, row)
		if d.RowsRemaining(ctx) == 0 {
			return nil, nil
		}
	}
	return nil, nil
}
