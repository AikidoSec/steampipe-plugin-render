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

// PostgresExport decorates the API export with the parent postgres ID. The
// `url` is a temporary signed download URL.
type PostgresExport struct {
	Id         string
	PostgresId string
	CreatedAt  time.Time
	Url        string
}

func tableRenderPostgresExport(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_postgres_export",
		Description: "A logical export of a Render Postgres database. The download URL is a temporary signed link.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderPostgres,
			Hydrate:       listRenderPostgresExports,
			KeyColumns:    plugin.OptionalColumns([]string{"postgres_id"}),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the export."},
			{Name: "postgres_id", Type: proto.ColumnType_STRING, Description: "The ID of the Postgres database the export was taken from."},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the export was created."},
			{Name: "url", Type: proto.ColumnType_STRING, Description: "Temporary signed URL to download the export, if still valid."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Id")},
		},
	}
}

func listRenderPostgresExports(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	pg := h.Item.(client.Postgres)

	if v := d.EqualsQualString("postgres_id"); v != "" && v != pg.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_postgres_export.listRenderPostgresExports", "connection_error", err)
		return nil, err
	}

	resp, err := c.ListPostgresExportWithResponse(ctx, pg.Id)
	if err != nil {
		logger.Error("render_postgres_export.listRenderPostgresExports", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("list postgres_exports failed: %s: %s", resp.Status(), string(resp.Body))
	}

	for _, ex := range *resp.JSON200 {
		row := PostgresExport{
			Id:         ex.Id,
			PostgresId: pg.Id,
			CreatedAt:  ex.CreatedAt,
		}
		if ex.Url != nil {
			row.Url = *ex.Url
		}
		d.StreamListItem(ctx, row)
		if d.RowsRemaining(ctx) == 0 {
			return nil, nil
		}
	}
	return nil, nil
}
