package render

import (
	"context"

	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

const pluginName = "steampipe-plugin-render"

// Plugin returns the Render Steampipe plugin definition.
func Plugin(ctx context.Context) *plugin.Plugin {
	p := &plugin.Plugin{
		Name:             pluginName,
		DefaultTransform: transform.FromCamel().Transform(transform.NullIfZeroValue),
		DefaultGetConfig: &plugin.GetConfig{},
		ConnectionConfigSchema: &plugin.ConnectionConfigSchema{
			NewInstance: ConfigInstance,
		},
		TableMap: map[string]*plugin.Table{
			"render_blueprint":             tableRenderBlueprint(ctx),
			"render_custom_domain":         tableRenderCustomDomain(ctx),
			"render_deploy":                tableRenderDeploy(ctx),
			"render_disk":                  tableRenderDisk(ctx),
			"render_env_group":             tableRenderEnvGroup(ctx),
			"render_environment":           tableRenderEnvironment(ctx),
			"render_header":                tableRenderHeader(ctx),
			"render_job":                   tableRenderJob(ctx),
			"render_key_value":             tableRenderKeyValue(ctx),
			"render_log_stream":            tableRenderLogStream(ctx),
			"render_notification_override": tableRenderNotificationOverride(ctx),
			"render_owner":                 tableRenderOwner(ctx),
			"render_postgres":              tableRenderPostgres(ctx),
			"render_postgres_export":       tableRenderPostgresExport(ctx),
			"render_project":               tableRenderProject(ctx),
			"render_registry_credential":   tableRenderRegistryCredential(ctx),
			"render_route":                 tableRenderRoute(ctx),
			"render_secret_file":           tableRenderSecretFile(ctx),
			"render_service":               tableRenderService(ctx),
			"render_snapshot":              tableRenderSnapshot(ctx),
			"render_webhook":               tableRenderWebhook(ctx),
		},
	}
	return p
}
