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

type CustomDomain struct {
	client.CustomDomain
	ServiceId string
}

func tableRenderCustomDomain(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_custom_domain",
		Description: "A custom domain attached to a Render web or static-site service.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderServices,
			Hydrate:       listRenderCustomDomains,
			KeyColumns: plugin.KeyColumnSlice{
				{Name: "service_id", Require: plugin.Optional},
				{Name: "name", Require: plugin.Optional},
				{Name: "domain_type", Require: plugin.Optional},
				{Name: "verification_status", Require: plugin.Optional},
			},
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderCustomDomain,
			KeyColumns: plugin.AllColumns([]string{"service_id", "id"}),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the custom domain."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The fully qualified domain name."},
			{Name: "service_id", Type: proto.ColumnType_STRING, Description: "The ID of the service this domain is attached to."},
			{Name: "domain_type", Type: proto.ColumnType_STRING, Description: "Whether this is an apex domain or subdomain."},
			{Name: "verification_status", Type: proto.ColumnType_STRING, Description: "Whether the domain is verified or unverified."},
			{Name: "public_suffix", Type: proto.ColumnType_STRING, Description: "The public suffix portion of the domain (e.g. example.com)."},
			{Name: "redirect_for_name", Type: proto.ColumnType_STRING, Description: "If this domain redirects to another, the destination domain name."},
			{Name: "server", Type: proto.ColumnType_JSON, Description: "Render's verification server info, if applicable."},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Description: "Time the domain was added."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderCustomDomains(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	service := h.Item.(client.Service)

	// Only web services and static sites can have custom domains. Asking the
	// API for any other service type returns 400 service not found, which would
	// kill the whole query when walking all services in a workspace.
	if service.Type != client.WebService && service.Type != client.StaticSite {
		return nil, nil
	}

	if v := d.EqualsQualString("service_id"); v != "" && v != service.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_custom_domain.listRenderCustomDomains", "connection_error", err)
		return nil, err
	}

	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListCustomDomainsParams{Limit: &limit}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &[]string{v}
	}
	if v := d.EqualsQualString("domain_type"); v != "" {
		t := client.ListCustomDomainsParamsDomainType(v)
		params.DomainType = &t
	}
	if v := d.EqualsQualString("verification_status"); v != "" {
		s := client.ListCustomDomainsParamsVerificationStatus(v)
		params.VerificationStatus = &s
	}

	for {
		resp, err := callWithRetry(ctx, func() (*client.ListCustomDomainsResponse, *http.Response, error) {
			r, e := c.ListCustomDomainsWithResponse(ctx, service.Id, params)
			if r != nil {
				return r, r.HTTPResponse, e
			}
			return r, nil, e
		})
		if err != nil {
			logger.Error("render_custom_domain.listRenderCustomDomains", "query_error", err)
			return nil, err
		}
		if resp.JSON200 == nil {
			if resp.StatusCode() == 404 && params.Cursor != nil {
				return nil, nil
			}
			return nil, fmt.Errorf("list custom_domains failed: %s: %s", resp.Status(), string(resp.Body))
		}

		page := *resp.JSON200
		if len(page) == 0 {
			return nil, nil
		}

		var lastCursor client.Cursor
		for _, item := range page {
			d.StreamListItem(ctx, CustomDomain{CustomDomain: item.CustomDomain, ServiceId: service.Id})
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

func getRenderCustomDomain(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	serviceID := d.EqualsQualString("service_id")
	id := d.EqualsQualString("id")
	if serviceID == "" || id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_custom_domain.getRenderCustomDomain", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveCustomDomainWithResponse(ctx, serviceID, id)
	if err != nil {
		logger.Error("render_custom_domain.getRenderCustomDomain", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve custom_domain failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return CustomDomain{CustomDomain: *resp.JSON200, ServiceId: serviceID}, nil
}
