package render

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

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

	apiClient, err := client.NewClientWithResponses(
		apiURL,
		client.WithRequestEditorFn(authEditor),
		client.WithRequestEditorFn(uaEditor),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating Render client: %w", err)
	}
	return apiClient, nil
}

// callWithRetry invokes op and retries on HTTP 429, respecting Retry-After when
// present and falling back to exponential backoff (capped at 30s). The closure
// returns the typed response, the underlying *http.Response (so we can read
// Retry-After), and any transport error. Used to soften the N-API-call burst
// from parent-hydrated tables that walk every service in a workspace.
func callWithRetry[T any](ctx context.Context, op func() (T, *http.Response, error)) (T, error) {
	const maxAttempts = 6
	var zero T
	delay := 500 * time.Millisecond
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, httpResp, err := op()
		if err != nil {
			return zero, err
		}
		if httpResp == nil || httpResp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}
		wait := delay
		if ra := httpResp.Header.Get("Retry-After"); ra != "" {
			if secs, perr := strconv.Atoi(ra); perr == nil && secs > 0 {
				wait = time.Duration(secs) * time.Second
			}
		}
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(wait):
		}
		delay *= 2
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
	}
	return zero, fmt.Errorf("rate limited after %d retries", maxAttempts)
}
