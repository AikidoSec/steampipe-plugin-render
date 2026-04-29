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

// Deploy decorates a client.Deploy with the parent service ID, since the API
// embeds deploys under /services/{serviceId}/deploys and doesn't echo the
// service ID in the deploy itself.
type Deploy struct {
	client.Deploy
	ServiceId string
}

func tableRenderDeploy(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_deploy",
		Description: "A deploy of a Render service. Listing requires a service_id (or implicit join through render_service).",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderServices,
			Hydrate:       listRenderDeploys,
			KeyColumns: plugin.KeyColumnSlice{
				{Name: "service_id", Require: plugin.Optional},
				{Name: "status", Require: plugin.Optional},
			},
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderDeploy,
			KeyColumns: plugin.AllColumns([]string{"service_id", "id"}),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the deploy."},
			{Name: "service_id", Type: proto.ColumnType_STRING, Description: "The ID of the service this deploy belongs to."},
			{Name: "status", Type: proto.ColumnType_STRING, Description: "The deploy status (e.g. live, build_failed, canceled)."},
			{Name: "trigger", Type: proto.ColumnType_STRING, Description: "What triggered this deploy."},
			{Name: "commit_id", Type: proto.ColumnType_STRING, Description: "The commit SHA the deploy was built from.", Transform: transform.FromField("Commit.Id")},
			{Name: "commit_message", Type: proto.ColumnType_STRING, Description: "The commit message for the deploy.", Transform: transform.FromField("Commit.Message")},
			{Name: "commit_created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the commit was created.", Transform: transform.FromField("Commit.CreatedAt")},
			{Name: "image_ref", Type: proto.ColumnType_STRING, Description: "Image reference used to build the deploy (image-deployed services only).", Transform: transform.FromField("Image.Ref")},
			{Name: "image_sha", Type: proto.ColumnType_STRING, Description: "SHA the image reference was resolved to (image-deployed services only).", Transform: transform.FromField("Image.Sha")},
			{Name: "image_registry_credential", Type: proto.ColumnType_STRING, Description: "Registry credential used to pull the image, if any.", Transform: transform.FromField("Image.RegistryCredential")},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the deploy was created."},
			{Name: "updated_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the deploy was last updated."},
			{Name: "finished_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the deploy finished, if applicable."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Id")},
		},
	}
}

func listRenderDeploys(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	service := h.Item.(client.Service)

	// Skip services that don't match the user's service_id qual.
	if v := d.EqualsQualString("service_id"); v != "" && v != service.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_deploy.listRenderDeploys", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListDeploysParams{Limit: &limit}
	if v := d.EqualsQualString("status"); v != "" {
		s := []client.DeployStatus{client.DeployStatus(v)}
		params.Status = &s
	}

	for {
		resp, err := callWithRetry(ctx, func() (*client.ListDeploysResponse, *http.Response, error) {
			r, e := c.ListDeploysWithResponse(ctx, service.Id, params)
			if r != nil {
				return r, r.HTTPResponse, e
			}
			return r, nil, e
		})
		if err != nil {
			logger.Error("render_deploy.listRenderDeploys", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list deploys failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			if item.Deploy == nil {
				continue
			}
			d.StreamListItem(ctx, Deploy{Deploy: *item.Deploy, ServiceId: service.Id})
			if item.Cursor != nil {
				lastCursor = *item.Cursor
			}
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

func getRenderDeploy(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	serviceID := d.EqualsQualString("service_id")
	id := d.EqualsQualString("id")
	if serviceID == "" || id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_deploy.getRenderDeploy", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveDeployWithResponse(ctx, serviceID, id)
	if err != nil {
		logger.Error("render_deploy.getRenderDeploy", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve deploy failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return Deploy{Deploy: *resp.JSON200, ServiceId: serviceID}, nil
}
