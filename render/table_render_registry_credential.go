package render

import (
	"context"
	"fmt"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

type RegistryCredential struct {
	client.RegistryCredential
	OwnerId string
}

func tableRenderRegistryCredential(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "render_registry_credential",
		Description: "A container-registry credential stored in a Render workspace and used to pull private images for image-deployed services.",
		List: &plugin.ListConfig{
			ParentHydrate: listRenderOwners,
			Hydrate:       listRenderRegistryCredentials,
			KeyColumns:    plugin.OptionalColumns([]string{"name", "username", "registry", "owner_id"}),
		},
		Get: &plugin.GetConfig{
			Hydrate:    getRenderRegistryCredential,
			KeyColumns: plugin.SingleColumn("id"),
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The unique identifier of the credential."},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The display name of the credential."},
			{Name: "registry", Type: proto.ColumnType_STRING, Description: "The registry this credential is for (e.g. dockerhub, ghcr, gar)."},
			{Name: "username", Type: proto.ColumnType_STRING, Description: "The username associated with the credential."},
			{Name: "updated_at", Type: proto.ColumnType_TIMESTAMP, Description: "Last updated time for the credential."},
			{Name: "owner_id", Type: proto.ColumnType_STRING, Description: "The ID of the workspace that owns this credential."},
			{Name: "title", Type: proto.ColumnType_STRING, Description: "Title of the resource.", Transform: transform.FromField("Name")},
		},
	}
}

func listRenderRegistryCredentials(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	owner := h.Item.(client.Owner)

	if v := d.EqualsQualString("owner_id"); v != "" && v != owner.Id {
		return nil, nil
	}

	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_registry_credential.listRenderRegistryCredentials", "connection_error", err)
		return nil, err
	}

	// The schema's response type for List is []RegistryCredential with no cursor,
	// so we make a single request at the largest supported page size.
	limit := defaultPageSize
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit > 0 && int(*d.QueryContext.Limit) < limit {
		limit = int(*d.QueryContext.Limit)
	}

	params := &client.ListRegistryCredentialsParams{
		Limit:   &limit,
		OwnerId: &client.OwnerIdParam{owner.Id},
	}
	if v := d.EqualsQualString("name"); v != "" {
		params.Name = &[]string{v}
	}
	if v := d.EqualsQualString("username"); v != "" {
		params.Username = &[]string{v}
	}
	if v := d.EqualsQualString("registry"); v != "" {
		t := []client.RegistryCredentialRegistry{client.RegistryCredentialRegistry(v)}
		params.Type = &t
	}

	resp, err := c.ListRegistryCredentialsWithResponse(ctx, params)
	if err != nil {
		logger.Error("render_registry_credential.listRenderRegistryCredentials", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list registry_credentials failed: %s: %s", resp.Status(), string(resp.Body))
	}

	for _, item := range *resp.JSON200 {
		d.StreamListItem(ctx, RegistryCredential{RegistryCredential: item, OwnerId: owner.Id})
		if d.RowsRemaining(ctx) == 0 {
			return nil, nil
		}
	}
	return nil, nil
}

func getRenderRegistryCredential(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	id := d.EqualsQualString("id")
	if id == "" {
		return nil, nil
	}

	logger := plugin.Logger(ctx)
	c, err := getClient(ctx, d)
	if err != nil {
		logger.Error("render_registry_credential.getRenderRegistryCredential", "connection_error", err)
		return nil, err
	}

	resp, err := c.RetrieveRegistryCredentialWithResponse(ctx, id)
	if err != nil {
		logger.Error("render_registry_credential.getRenderRegistryCredential", "query_error", err)
		return nil, err
	}
	if resp.JSON200 == nil {
		if resp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("retrieve registry_credential failed: %s: %s", resp.Status(), string(resp.Body))
	}
	return *resp.JSON200, nil
}
