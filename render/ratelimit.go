package render

import (
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Based on https://api-docs.render.com/reference/rate-limiting
const (
	generalGetPerMin = 380
	generalGetBurst  = 10

	logsGetPerMin = 28
	logsGetBurst  = 2
)

const (
	maxAttempts    = 6
	initialBackoff = 500 * time.Millisecond
	maxBackoff     = 30 * time.Second
	retryJitter    = time.Second
)

var limiterRegistry sync.Map

type endpointLimiters struct {
	general *rate.Limiter
	logs    *rate.Limiter
}

func (e *endpointLimiters) forRequest(req *http.Request) *rate.Limiter {
	if strings.HasSuffix(req.URL.Path, "/logs") {
		return e.logs
	}
	return e.general
}

func limitersForKey(apiKey string) *endpointLimiters {
	if existing, ok := limiterRegistry.Load(apiKey); ok {
		return existing.(*endpointLimiters)
	}
	created := &endpointLimiters{
		general: rate.NewLimiter(perMinute(generalGetPerMin), generalGetBurst),
		logs:    rate.NewLimiter(perMinute(logsGetPerMin), logsGetBurst),
	}
	actual, _ := limiterRegistry.LoadOrStore(apiKey, created)
	return actual.(*endpointLimiters)
}

func perMinute(n int) rate.Limit {
	return rate.Limit(float64(n) / 60.0)
}

type rateLimitTransport struct {
	base     http.RoundTripper
	limiters *endpointLimiters
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	limiter := t.limiters.forRequest(req)
	retryable := req.Body == nil || req.Body == http.NoBody
	backoff := initialBackoff
	for attempt := 0; ; attempt++ {
		if err := limiter.Wait(req.Context()); err != nil {
			return nil, err
		}

		resp, err := t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests || !retryable || attempt >= maxAttempts-1 {
			return resp, nil
		}

		wait := retryWait(resp, backoff)
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func retryWait(resp *http.Response, backoff time.Duration) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
			return time.Duration(secs)*time.Second + jitterUpTo(retryJitter)
		}
	}
	if reset := resp.Header.Get("Ratelimit-Reset"); reset != "" {
		if epoch, err := strconv.ParseInt(reset, 10, 64); err == nil {
			if d := time.Until(time.Unix(epoch, 0)); d > 0 {
				return d + jitterUpTo(retryJitter)
			}
		}
	}
	return backoff + jitterUpTo(backoff/2)
}

func jitterUpTo(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(max)))
}
