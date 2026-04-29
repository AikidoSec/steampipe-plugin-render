package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tableRenderProject(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_project",
		Description: "A Render project. Projects group together environments and the resources that belong to them.",
		List: &plugin.ListConfig{
			Hydrate:    listRenderProjects,
			KeyColumns: plugin.OptionalColumns([]string{"name", "owner_id"}),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderProject,
			KeyColumns: plugin.SingleColumn("id"),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the project."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the project."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace that owns this project.", Transform: transform.FromField("Owner.Id")},
			{Name: "owner", Type: proto.ColumnType_JSON, Description: "Summary of the workspace that owns this project."},
			{Name: "environment_ids", Type: proto.ColumnType_JSON, Description: "IDs of the environments that belong to this project."},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the project was created."},
			{Name: "updated_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the project was last updated."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderProjects(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_project.listRenderProjects", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListProjectsParams{Limit: &limit}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &client.NameParam{v}
	}
	if v := d.EqualsQualString("owner_id"); v != "" {
		params.OwnerId = &client.OwnerIdParam{v}
	}

	for {
		resp, err := c.ListProjectsWithResponse(ctx, params)
		if err != nil {
			logger.Error("render_project.listRenderProjects", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list projects failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, item.Project)
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

func getRenderProject(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	id := d.EqualsQualString("id")
	if id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_project.getRenderProject", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveProjectWithResponse(ctx, id)
	if err != nil {
		logger.Error("render_project.getRenderProject", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve project failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return *resp.JSON200, nil
}
