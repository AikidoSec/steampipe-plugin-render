package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/render-oss/steampipe-plugin-render/render/client/blueprints"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

type Blueprint struct {
	blueprints.Blueprint
	OwnerId string
}

func tableRenderBlueprint(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_blueprint",
		Description: "A Render blueprint (render.yaml-driven IaC definition).",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderOwners,
			Hydrate:       listRenderBlueprints,
			KeyColumns:    plugin.OptionalColumns([]string{"owner_id"}),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderBlueprint,
			KeyColumns: plugin.SingleColumn("id"),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the blueprint."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the blueprint."},
			{Name: "repo", Type: proto.ColumnType_STRING, Description: "The git repository URL the blueprint is sourced from."},
			{Name: "branch", Type: proto.ColumnType_STRING, Description: "The git branch the blueprint syncs from."},
			{Name: "auto_sync", Type: proto.ColumnType_BOOL, Description: "Whether render.yaml changes are automatically synced."},
			{Name: "status", Type: proto.ColumnType_STRING, Description: "Sync status of the blueprint (in_sync, syncing, error, paused, created)."},
			{Name: "last_sync", Type: proto.ColumnType_TIMESTAMP, Description: "Time the blueprint was last successfully synced."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace that owns this blueprint."},
			// resources is only populated by Retrieve, not List. Hydrate it lazily.
			{Name: "resources", Type: proto.ColumnType_JSON, Description: "Resources defined by the blueprint. Hydrated per row from the Retrieve endpoint.", Hydrate: getBlueprintResources, Transform: transform.FromField("Resources")},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderBlueprints(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	owner := h.Item.(client.Owner)

	if v := d.EqualsQualString("owner_id"); v != "" && v != owner.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_blueprint.listRenderBlueprints", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListBlueprintsParams{
		Limit:   &limit,
		OwnerId: &client.OwnerIdParam{owner.Id},
	}

	for {
		resp, err := c.ListBlueprintsWithResponse(ctx, params)
		if err != nil {
			logger.Error("render_blueprint.listRenderBlueprints", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list blueprints failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, Blueprint{Blueprint: item.Blueprint, OwnerId: owner.Id})
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

func getRenderBlueprint(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	id := d.EqualsQualString("id")
	if id == "" {
		return nil, nil
	}
	return fetchBlueprint(ctx, d, id)
}

// getBlueprintResources hydrates the `resources` column. Only invoked when
// `resources` is part of the SELECT.
func getBlueprintResources(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	switch v := h.Item.(type) {
	case blueprints.BlueprintDetail:
		return v, nil
	case Blueprint:
		return fetchBlueprint(ctx, d, v.Id)
	case blueprints.Blueprint:
		return fetchBlueprint(ctx, d, v.Id)
	default:
		return nil, fmt.Errorf("unexpected blueprint row type %T", h.Item)
	}
}

func fetchBlueprint(ctx context.Context, d *plugin.QueryData, id string) (interface{}, error) {
	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_blueprint.fetchBlueprint", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveBlueprintWithResponse(ctx, id)
	if err != nil {
		logger.Error("render_blueprint.fetchBlueprint", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve blueprint failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return *resp.JSON200, nil
}
