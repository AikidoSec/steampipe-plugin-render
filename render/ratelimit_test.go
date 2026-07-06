package render

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestForRequest(t *testing.T) {
	limiters := &endpointLimiters{
		general: rate.NewLimiter(1, 1),
		logs:    rate.NewLimiter(1, 1),
	}

	cases := []struct {
		name   string
		method string
		url    string
		want   *rate.Limiter
	}{
		{"general GET", http.MethodGet, "https://api.render.com/v1/services", limiters.general},
		{"logs search GET", http.MethodGet, "https://api.render.com/v1/logs", limiters.logs},
		{"log stream config is general", http.MethodGet, "https://api.render.com/v1/logs/streams/owner/abc", limiters.general},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatalf("building request: %v", err)
			}
			if got := limiters.forRequest(req); got != tc.want {
				t.Errorf("forRequest(%s %s) returned the wrong bucket", tc.method, tc.url)
			}
		})
	}
}

func respWithHeaders(headers map[string]string) *http.Response {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{Header: h}
}

func TestRetryWait(t *testing.T) {
	const backoff = 500 * time.Millisecond

	t.Run("Retry-After wins", func(t *testing.T) {
		resp := respWithHeaders(map[string]string{
			"Retry-After":     "3",
			"Ratelimit-Reset": "9999999999",
		})
		// 3s from the header plus up to retryJitter of jitter.
		if got := retryWait(resp, backoff); got < 3*time.Second || got >= 3*time.Second+retryJitter {
			t.Errorf("got %v, want [3s, 3s+%v)", got, retryJitter)
		}
	})

	t.Run("Ratelimit-Reset used when Retry-After absent", func(t *testing.T) {
		reset := time.Now().Add(5 * time.Second).Unix()
		resp := respWithHeaders(map[string]string{
			"Ratelimit-Reset": strconv.FormatInt(reset, 10),
		})
		got := retryWait(resp, backoff)
		// Wait until the reset epoch (≤5s after truncation) plus up to retryJitter.
		if got <= 3*time.Second || got > 5*time.Second+retryJitter {
			t.Errorf("got %v, want between 3s and 5s+%v", got, retryJitter)
		}
	})

	t.Run("invalid Retry-After falls through", func(t *testing.T) {
		resp := respWithHeaders(map[string]string{"Retry-After": "0"})
		got := retryWait(resp, backoff)
		if got < backoff || got > backoff+backoff/2 {
			t.Errorf("got %v, want backoff+jitter in [%v, %v]", got, backoff, backoff+backoff/2)
		}
	})

	t.Run("no headers uses backoff with jitter", func(t *testing.T) {
		resp := respWithHeaders(nil)
		got := retryWait(resp, backoff)
		if got < backoff || got > backoff+backoff/2 {
			t.Errorf("got %v, want backoff+jitter in [%v, %v]", got, backoff, backoff+backoff/2)
		}
	})

	t.Run("past Ratelimit-Reset falls through to backoff", func(t *testing.T) {
		reset := time.Now().Add(-10 * time.Second).Unix()
		resp := respWithHeaders(map[string]string{"Ratelimit-Reset": strconv.FormatInt(reset, 10)})
		got := retryWait(resp, backoff)
		if got < backoff || got > backoff+backoff/2 {
			t.Errorf("got %v, want backoff+jitter in [%v, %v]", got, backoff, backoff+backoff/2)
		}
	})
}
