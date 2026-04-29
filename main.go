package main

import (
	"github.com/render-oss/steampipe-plugin-render/render"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		PluginFunc: render.Plugin,
	})
}
