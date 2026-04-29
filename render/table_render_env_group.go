package render

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tableRenderEnvGroup(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_env_group",
		Description: "A shared group of environment variables (and optionally secret files) that can be linked to multiple Render services. This table exposes metadata only and does not retrieve secret-bearing payloads.",
		List: &plugin.ListConfig{
			Hydrate:    listRenderEnvGroups,
			KeyColumns: plugin.OptionalColumns([]string{"name", "owner_id", "environment_id"}),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the env group."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the env group."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace that owns the env group."},
			{Name: "environment_id", Type: proto.ColumnType_STRING, Description: "The ID of the environment the env group is scoped to, if any."},
			{Name: "service_links", Type: proto.ColumnType_JSON, Description: "Services this env group is linked to."},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the env group was created."},
			{Name: "updated_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the env group was last updated."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderEnvGroups(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_env_group.listRenderEnvGroups", "connection_error", err)
		return nil, err
	}

	// The schema's ListEnvGroups response does not expose a cursor, so we
	// make a single request with the largest supported page size.
	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}
	params := &client.ListEnvGroupsParams{Limit: &limit}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &client.NameParam{v}
	}
	if v := d.EqualsQualString("owner_id"); v != "" {
		params.OwnerId = &client.OwnerIdParam{v}
	}
	if v := d.EqualsQualString("environment_id"); v != "" {
		params.EnvironmentId = &client.EnvironmentIdParam{v}
	}

	resp, err := c.ListEnvGroupsWithResponse(ctx, params)
	if err != nil {
		logger.Error("render_env_group.listRenderEnvGroups", "query_error", err)
		return nil, err
	}
	if resp.HTTPResponse == nil || resp.HTTPResponse.StatusCode != 200 {
		return nil, fmt.Errorf("list env_groups failed: %s: %s", resp.Status(), string(resp.Body))
	}

	// The OpenAPI schema declares this response as []EnvGroupMeta but the API
	// actually returns [{"cursor": "...", "envGroup": {...}}, ...]. We can't
	// trust the codegen-decoded JSON200 (it produces zero-valued items); decode
	// the raw body ourselves.
	var wrappers []struct {
		Cursor   string              `json:"cursor"`
		EnvGroup client.EnvGroupMeta `json:"envGroup"`
	}
	if err := json.Unmarshal(resp.Body, &wrappers); err != nil {
		logger.Error("render_env_group.listRenderEnvGroups", "decode_error", err)
		return nil, fmt.Errorf("decode env_groups response: %w", err)
	}

	for _, w := range wrappers {
		d.StreamListItem(ctx, w.EnvGroup)
		if d.RowsRemaining(ctx) == 0 {
			return nil, nil
		}
	}
	return nil, nil
}
