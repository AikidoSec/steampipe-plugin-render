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

func tableRenderService(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_service",
		Description: "A service running on Render (web service, private service, background worker, cron job, or static site).",
		List: &plugin.ListConfig{
			Hydrate:    listRenderServices,
			KeyColumns: plugin.OptionalColumns([]string{"name", "type", "environment_id", "owner_id", "suspended"}),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderService,
			KeyColumns: plugin.SingleColumn("id"),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the service."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the service."},
			{Name: "slug", Type: proto.ColumnType_STRING, Description: "The slug of the service (used in dashboard URLs)."},
			{Name: "type", Type: proto.ColumnType_STRING, Description: "The type of service (e.g. web_service, private_service, background_worker, cron_job, static_site)."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace that owns this service."},
			{Name: "environment_id", Type: proto.ColumnType_STRING, Description: "The ID of the environment this service belongs to, if any."},
			{Name: "repo", Type: proto.ColumnType_STRING, Description: "The git repository URL the service is built from."},
			{Name: "branch", Type: proto.ColumnType_STRING, Description: "The git branch the service deploys from."},
			{Name: "auto_deploy", Type: proto.ColumnType_STRING, Description: "Whether autodeploy is enabled (yes / no)."},
			{Name: "auto_deploy_trigger", Type: proto.ColumnType_STRING, Description: "What triggers autodeploys (commit or checksPass)."},
			{Name: "image_path", Type: proto.ColumnType_STRING, Description: "The container image path, for image-deployed services."},
			{Name: "root_dir", Type: proto.ColumnType_STRING, Description: "The root directory used for builds within the repo."},
			{Name: "suspended", Type: proto.ColumnType_STRING, Description: "Whether the service is suspended (suspended / not_suspended)."},
			{Name: "suspenders", Type: proto.ColumnType_JSON, Description: "List of reasons the service has been suspended, if any."},
			{Name: "notify_on_fail", Type: proto.ColumnType_STRING, Description: "Notification preference for failed deploys."},
			{Name: "dashboard_url", Type: proto.ColumnType_STRING, Description: "URL to view the service in the Render Dashboard."},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the service was created."},
			{Name: "updated_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the service was last updated."},
			{Name: "service_details", Type: proto.ColumnType_JSON, Description: "Type-specific service details (region, plan, runtime, env, etc.)."},
			{Name: "build_filter", Type: proto.ColumnType_JSON, Description: "Build filter (paths/ignoredPaths) used to gate autodeploys."},
			{Name: "registry_credential", Type: proto.ColumnType_JSON, Description: "Container registry credential summary, for image-deployed services."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

const defaultPageSize = 100

func listRenderServices(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_service.listRenderServices", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListServicesParams{
		Limit: &limit,
	}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &client.NameParam{v}
	}
	if v := d.EqualsQualString("type"); v != "" {
		params.Type = &client.ServiceTypeParam{client.ServiceType(v)}
	}
	if v := d.EqualsQualString("environment_id"); v != "" {
		params.EnvironmentId = &client.EnvironmentIdParam{v}
	}
	if v := d.EqualsQualString("owner_id"); v != "" {
		params.OwnerId = &client.OwnerIdParam{v}
	}
	if v := d.EqualsQualString("suspended"); v != "" {
		s := client.SuspendedParam{v}
		params.Suspended = &s
	}

	for {
		// listRenderServices is the parent hydrate for several other tables
		// (deploy, custom_domain, header, ...), so a 429 here cascades and
		// breaks the whole query. Retry on 429 to soften the burst.
		resp, err := callWithRetry(ctx, func() (*client.ListServicesResponse, *http.Response, error) {
			r, e := c.ListServicesWithResponse(ctx, params)
			if r != nil {
				return r, r.HTTPResponse, e
			}
			return r, nil, e
		})
		if err != nil {
			logger.Error("render_service.listRenderServices", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			// Render's list endpoints sometimes return 404 when paginating past
			// the last available page. Treat that as end-of-results when we
			// already have a cursor; otherwise it's a real error.
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list services failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, item.Service)
			lastCursor = item.Cursor
			if d.RowsRemaining(ctx) == 0 {
				return nil, nil
			}
		}

		// If we got fewer than the requested page size, we're done.
		if len(page) < limit {
			return nil, nil
		}
		params.Cursor = &lastCursor
	}
}

func getRenderService(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	id := d.EqualsQualString("id")
	if id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_service.getRenderService", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveServiceWithResponse(ctx, id)
	if err != nil {
		logger.Error("render_service.getRenderService", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve service failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return *resp.JSON200, nil
}
