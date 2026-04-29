package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/render-oss/steampipe-plugin-render/render/client/webhooks"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// Webhook decorates the API webhook with the parent owner ID. We deliberately
// do not surface the signing secret as a column even though the API returns it.
type Webhook struct {
	Id          string
	Name        string
	Url         string
	Enabled     bool
	EventFilter webhooks.EventFilter
	OwnerId     string
}

func tableRenderWebhook(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_webhook",
		Description: "An outbound webhook configured for a Render workspace. The HMAC signing secret is deliberately not exposed.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderOwners,
			Hydrate:       listRenderWebhooks,
			KeyColumns:    plugin.OptionalColumns([]string{"owner_id"}),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderWebhook,
			KeyColumns: plugin.SingleColumn("id"),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the webhook."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the webhook."},
			{Name: "url", Type: proto.ColumnType_STRING, Description: "The destination URL events are POSTed to."},
			{Name: "enabled", Type: proto.ColumnType_BOOL, Description: "Whether the webhook is currently active."},
			{Name: "event_filter", Type: proto.ColumnType_JSON, Description: "Event types that will trigger this webhook. Empty means all events."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace that owns this webhook."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderWebhooks(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	owner := h.Item.(client.Owner)

	if v := d.EqualsQualString("owner_id"); v != "" && v != owner.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_webhook.listRenderWebhooks", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}
	params := &client.ListWebhooksParams{
		Limit:   &limit,
		OwnerId: &client.OwnerIdParam{owner.Id},
	}

	for {
		resp, err := c.ListWebhooksWithResponse(ctx, params)
		if err != nil {
			logger.Error("render_webhook.listRenderWebhooks", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list webhooks failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, Webhook{
				Id:          item.Webhook.Id,
				Name:        item.Webhook.Name,
				Url:         item.Webhook.Url,
				Enabled:     item.Webhook.Enabled,
				EventFilter: item.Webhook.EventFilter,
				OwnerId:     owner.Id,
			})
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

func getRenderWebhook(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	id := d.EqualsQualString("id")
	if id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_webhook.getRenderWebhook", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveWebhookWithResponse(ctx, id)
	if err != nil {
		logger.Error("render_webhook.getRenderWebhook", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve webhook failed: %s: %s", resp.Status(), string(resp.Body))
	}
	wh := *resp.JSON200
	return Webhook{
		Id:          wh.Id,
		Name:        wh.Name,
		Url:         wh.Url,
		Enabled:     wh.Enabled,
		EventFilter: wh.EventFilter,
	}, nil
}
