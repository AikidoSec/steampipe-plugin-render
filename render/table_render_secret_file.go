package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// SecretFile is the metadata about a secret file mounted in a service.
// Note: we deliberately do not surface the file content as a column even
// though the API includes it on the SecretFile type. Anyone needing the
// content can use the Render API directly.
type SecretFile struct {
	Name      string
	ServiceId string
}

func tableRenderSecretFile(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_secret_file",
		Description: "Names of secret files mounted into a Render service. File contents are deliberately not exposed.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderServices,
			Hydrate:       listRenderSecretFiles,
			KeyColumns: plugin.KeyColumnSlice{
				{Name: "service_id", Require: plugin.Optional},
			},
		},
		Columns: []*plugin.Column{
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The name (path) of the secret file."},
			{Name: "service_id", Type: proto.ColumnType_STRING, Description: "The ID of the service the file is mounted into."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderSecretFiles(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	service := h.Item.(client.Service)

	if v := d.EqualsQualString("service_id"); v != "" && v != service.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_secret_file.listRenderSecretFiles", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}
	params := &client.ListSecretFilesForServiceParams{Limit: &limit}

	for {
		resp, err := c.ListSecretFilesForServiceWithResponse(ctx, service.Id, params)
		if err != nil {
			logger.Error("render_secret_file.listRenderSecretFiles", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list secret_files failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, SecretFile{Name: item.SecretFile.Name, ServiceId: service.Id})
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
