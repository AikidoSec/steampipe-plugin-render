package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/render-oss/steampipe-plugin-render/render/client/jobs"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tableRenderJob(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_job",
		Description: "A one-off job (background script) run against a Render service.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderServices,
			Hydrate:       listRenderJobs,
			KeyColumns: plugin.KeyColumnSlice{
				{Name: "service_id", Require: plugin.Optional},
				{Name: "status", Require: plugin.Optional},
			},
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderJob,
			KeyColumns: plugin.AllColumns([]string{"service_id", "id"}),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the job."},
			{Name: "service_id", Type: proto.ColumnType_STRING, Description: "The ID of the service the job ran against."},
			{Name: "status", Type: proto.ColumnType_STRING, Description: "The job status (pending, running, succeeded, failed, canceled)."},
			{Name: "start_command", Type: proto.ColumnType_STRING, Description: "The command the job runs."},
			{Name: "plan_id", Type: proto.ColumnType_STRING, Description: "The ID of the plan/instance type the job runs on."},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the job was created."},
			{Name: "started_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the job started running."},
			{Name: "finished_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the job finished, if applicable."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Id")},
		},
	}
}

func listRenderJobs(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	service := h.Item.(client.Service)

	if v := d.EqualsQualString("service_id"); v != "" && v != service.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_job.listRenderJobs", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListJobParams{Limit: &limit}
	if v := d.EqualsQualString("status"); v != "" {
		s := []jobs.JobStatus{jobs.JobStatus(v)}
		params.Status = &s
	}

	for {
		resp, err := c.ListJobWithResponse(ctx, service.Id, params)
		if err != nil {
			logger.Error("render_job.listRenderJobs", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list jobs failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, item.Job)
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

func getRenderJob(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	serviceID := d.EqualsQualString("service_id")
	id := d.EqualsQualString("id")
	if serviceID == "" || id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_job.getRenderJob", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveJobWithResponse(ctx, serviceID, id)
	if err != nil {
		logger.Error("render_job.getRenderJob", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve job failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return *resp.JSON200, nil
}
