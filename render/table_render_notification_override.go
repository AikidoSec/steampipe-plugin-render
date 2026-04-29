package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/render-oss/steampipe-plugin-render/render/client/notifications"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

type NotificationOverride struct {
	notifications.NotificationOverride
	OwnerId string
}

func tableRenderNotificationOverride(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_notification_override",
		Description: "Per-service overrides of the workspace-level notification settings.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderOwners,
			Hydrate:       listRenderNotificationOverrides,
			KeyColumns:    plugin.OptionalColumns([]string{"service_id", "owner_id"}),
		},
		Columns: []*plugin.Column{
			{Name: "service_id", Type: proto.ColumnType_STRING, Description: "The ID of the service this override applies to."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace this override belongs to."},
			{Name: "notifications_to_send", Type: proto.ColumnType_STRING, Description: "Override for which deploy outcomes trigger notifications (default, all, failure, none)."},
			{Name: "preview_notifications_enabled", Type: proto.ColumnType_STRING, Description: "Override for preview-environment notifications (default, true, false)."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("ServiceId")},
		},
	}
}

func listRenderNotificationOverrides(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	owner := h.Item.(client.Owner)

	if v := d.EqualsQualString("owner_id"); v != "" && v != owner.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_notification_override.listRenderNotificationOverrides", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListNotificationOverridesParams{
		Limit:   &limit,
		OwnerId: &client.OwnerIdParam{owner.Id},
	}
	if v := d.EqualsQualString("service_id"); v != "" {
		params.ServiceId = &client.ServiceIdsParam{v}
	}

	for {
		resp, err := c.ListNotificationOverridesWithResponse(ctx, params)
		if err != nil {
			logger.Error("render_notification_override.listRenderNotificationOverrides", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list notification_overrides failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, NotificationOverride{NotificationOverride: item.Override, OwnerId: owner.Id})
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
