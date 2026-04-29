package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tableRenderEnvironment(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_environment",
		Description: "An environment within a Render project. Environments group resources and can be marked protected.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderProjects,
			Hydrate:       listRenderEnvironments,
			KeyColumns:    plugin.OptionalColumns([]string{"project_id", "name"}),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderEnvironment,
			KeyColumns: plugin.SingleColumn("id"),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the environment."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the environment."},
			{Name: "project_id", Type: proto.ColumnType_STRING, Description: "The ID of the project this environment belongs to."},
			{Name: "protected_status", Type: proto.ColumnType_STRING, Description: "Whether the environment is 'protected' or 'unprotected'. Only admins can perform destructive actions in protected environments."},
			{Name: "network_isolation_enabled", Type: proto.ColumnType_BOOL, Description: "Whether network connections across environments are blocked."},
			{Name: "service_ids", Type: proto.ColumnType_JSON, Description: "IDs of services in this environment."},
			{Name: "databases_ids", Type: proto.ColumnType_JSON, Description: "IDs of Postgres databases in this environment."},
			{Name: "redis_ids", Type: proto.ColumnType_JSON, Description: "IDs of Redis / key-value instances in this environment."},
			{Name: "env_group_ids", Type: proto.ColumnType_JSON, Description: "IDs of env groups associated with this environment."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderEnvironments(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	project := h.Item.(client.Project)

	// If the user filtered by project_id, skip projects that don't match
	// rather than making a wasted API call per-project.
	if v := d.EqualsQualString("project_id"); v != "" && v != project.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_environment.listRenderEnvironments", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListEnvironmentsParams{
		ProjectId: client.ProjectIdParam{project.Id},
		Limit:     &limit,
	}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &client.NameParam{v}
	}

	for {
		resp, err := c.ListEnvironmentsWithResponse(ctx, params)
		if err != nil {
			logger.Error("render_environment.listRenderEnvironments", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list environments failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, item.Environment)
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

func getRenderEnvironment(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	id := d.EqualsQualString("id")
	if id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_environment.getRenderEnvironment", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveEnvironmentWithResponse(ctx, id)
	if err != nil {
		logger.Error("render_environment.getRenderEnvironment", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve environment failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return *resp.JSON200, nil
}
