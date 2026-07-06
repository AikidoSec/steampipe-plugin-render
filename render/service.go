package render

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/render-oss/steampipe-plugin-render/render/client"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

const defaultAPIURL = "https://api.render.com/v1"

// getClient returns a Render API client. The result is memoized per connection
// so we only build it once per query.
func getClient(ctx context.Context, d *plugin.QueryData) (*client.ClientWithResponses, error) {
	conn, err := clientCached(ctx, d, nil)
	if err != nil {
		return nil, err
	}
	return conn.(*client.ClientWithResponses), nil
}

var clientCached = plugin.HydrateFunc(clientUncached).Memoize()

func clientUncached(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (any, error) {
	cfg := GetConfig(d.Connection)

	// Credential precedence: connection config > RENDER_API_KEY env var.
	apiKey := os.Getenv("RENDER_API_KEY")
	if cfg.APIKey != nil {
		apiKey = *cfg.APIKey
	}
	if apiKey == "" {
		return nil, fmt.Errorf("api_key must be configured (or set RENDER_API_KEY)")
	}

	apiURL := defaultAPIURL
	if cfg.APIURL != nil && *cfg.APIURL != "" {
		apiURL = *cfg.APIURL
	}

	authEditor := func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Accept", "application/json")
		return nil
	}
	uaEditor := func(_ context.Context, req *http.Request) error {
		req.Header.Set("User-Agent", "steampipe-plugin-render")
		return nil
	}

	httpClient := &http.Client{
		Transport: &rateLimitTransport{
			base:     http.DefaultTransport,
			limiters: limitersForKey(apiKey),
		},
	}

	apiClient, err := client.NewClientWithResponses(
		apiURL,
		client.WithHTTPClient(httpClient),
		client.WithRequestEditorFn(authEditor),
		client.WithRequestEditorFn(uaEditor),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating Render client: %w", err)
	}
	return apiClient, nil
}
