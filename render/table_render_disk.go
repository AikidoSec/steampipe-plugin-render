package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/render-oss/steampipe-plugin-render/render/client/disks"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

type Disk struct {
	disks.DiskDetails
	OwnerId string
}

func tableRenderDisk(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_disk",
		Description: "A persistent disk attached to a Render service.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderOwners,
			Hydrate:       listRenderDisks,
			KeyColumns:    plugin.OptionalColumns([]string{"name", "owner_id", "service_id"}),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderDisk,
			KeyColumns: plugin.SingleColumn("id"),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the disk."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the disk."},
			{Name: "service_id", Type: proto.ColumnType_STRING, Description: "ID of the service this disk is attached to, if any."},
			{Name: "mount_path", Type: proto.ColumnType_STRING, Description: "Filesystem path the disk is mounted at."},
			{Name: "size_gb", Type: proto.ColumnType_INT, Description: "Provisioned size in gigabytes.", Transform: transform.FromField("SizeGB")},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the disk was created."},
			{Name: "updated_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the disk was last updated."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace that owns this disk."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderDisks(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	owner := h.Item.(client.Owner)

	if v := d.EqualsQualString("owner_id"); v != "" && v != owner.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_disk.listRenderDisks", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListDisksParams{
		Limit:   &limit,
		OwnerId: &client.OwnerIdParam{owner.Id},
	}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &client.NameParam{v}
	}
	if v := d.EqualsQualString("service_id"); v != "" {
		params.ServiceId = &client.ServiceIdsParam{v}
	}

	for {
		resp, err := c.ListDisksWithResponse(ctx, params)
		if err != nil {
			logger.Error("render_disk.listRenderDisks", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list disks failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, Disk{DiskDetails: item.Disk, OwnerId: owner.Id})
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

func getRenderDisk(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	id := d.EqualsQualString("id")
	if id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_disk.getRenderDisk", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveDiskWithResponse(ctx, disks.DiskId(id))
	if err != nil {
		logger.Error("render_disk.getRenderDisk", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve disk failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return *resp.JSON200, nil
}
