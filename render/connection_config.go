package render

import (
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

type renderConfig struct {
	APIKey *string `hcl:"api_key"`
	APIURL *string `hcl:"api_url"`
}

func ConfigInstance() interface{} {
	return &renderConfig{}
}

// GetConfig retrieves and casts connection config from query data.
func GetConfig(connection *plugin.Connection) renderConfig {
	if connection == nil || connection.Config == nil {
		return renderConfig{}
	}
	config, _ := connection.Config.(renderConfig)
	return config
}
