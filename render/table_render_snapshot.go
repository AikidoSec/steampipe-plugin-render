package render

import (
	"context"
	"fmt"
	"time"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// Snapshot decorates a DiskSnapshot with the parent disk + owner IDs so we can
// join cleanly with render_disk and render_owner.
type Snapshot struct {
	SnapshotKey string
	DiskId      string
	OwnerId     string
	InstanceId  string
	CreatedAt   *time.Time
}

func tableRenderSnapshot(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_snapshot",
		Description: "A point-in-time snapshot of a disk attached to a Render service.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderOwners,
			Hydrate:       listRenderSnapshots,
			KeyColumns:    plugin.OptionalColumns([]string{"disk_id", "owner_id"}),
		},
		Columns: []*plugin.Column{
			{Name: "snapshot_key", Type: proto.ColumnType_STRING, Description: "The unique key for the snapshot."},
			{Name: "disk_id", Type: proto.ColumnType_STRING, Description: "The ID of the disk this snapshot belongs to."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace that owns the disk."},
			{Name: "instance_id", Type: proto.ColumnType_STRING, Description: "Instance ID, when the disk was attached to a scaled service."},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the snapshot was created."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("SnapshotKey")},
		},
	}
}

// listRenderSnapshots walks owner -> disks -> snapshots. Snapshots are only
// addressable per-disk in the API, so without a disk_id qual we have to
// enumerate disks for each owner. Filters are pushed through where possible.
func listRenderSnapshots(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	owner := h.Item.(client.Owner)

	if v := d.EqualsQualString("owner_id"); v != "" && v != owner.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_snapshot.listRenderSnapshots", "connection_error", err)
		return nil, err
	}

	diskQual := d.EqualsQualString("disk_id")

	// Step 1: list disks for this owner. Single page of up to 100 — re-use the
	// disks list endpoint with a cursor loop in case the workspace has more.
	diskLimit := defaultPageSize
	diskParams := &client.ListDisksParams{
		Limit:   &diskLimit,
		OwnerId: &client.OwnerIdParam{owner.Id},
	}

	for {
		diskResp, err := c.ListDisksWithResponse(ctx, diskParams)
		if err != nil {
			logger.Error("render_snapshot.listRenderSnapshots", "list_disks_error", err)
			return nil, err
		}
		if diskResp.JSON200 == nil {
			if diskResp.StatusCode() == 404 && diskParams.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list disks for snapshots failed: %s: %s", diskResp.Status(), string(diskResp.Body))
		}

		diskPage := *diskResp.JSON200
		if len(diskPage) == 0 {
			return nil, nil
		}

		var lastDiskCursor client.Cursor
		for _, dItem := range diskPage {
			lastDiskCursor = dItem.Cursor
			disk := dItem.Disk
			if diskQual != "" && diskQual != disk.Id {
				continue
			}

			// Step 2: list snapshots for this disk.
			snapResp, err := c.ListSnapshotsWithResponse(ctx, disk.Id)
			if err != nil {
				logger.Error("render_snapshot.listRenderSnapshots", "list_snapshots_error", err, "disk_id", disk.Id)
				return nil, err
			}
			if snapResp.JSON201 == nil {
				if snapResp.StatusCode() == 404 {
					continue // disk has no snapshots; keep going
				}
				return nil, fmt.Errorf("list snapshots for disk %s failed: %s: %s", disk.Id, snapResp.Status(), string(snapResp.Body))
			}

			for _, snap := range *snapResp.JSON201 {
				row := Snapshot{
					DiskId:    disk.Id,
					OwnerId:   owner.Id,
					CreatedAt: snap.CreatedAt,
				}
				if snap.SnapshotKey != nil {
					row.SnapshotKey = *snap.SnapshotKey
				}
				if snap.InstanceId != nil {
					row.InstanceId = *snap.InstanceId
				}
				d.StreamListItem(ctx, row)
				if d.RowsRemaining(ctx) == 0 {
					return nil, nil
				}
			}
		}

		if len(diskPage) < diskLimit {
			return nil, nil
		}
		diskParams.Cursor = &lastDiskCursor
	}
}
