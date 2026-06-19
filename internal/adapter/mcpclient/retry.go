package mcpclient

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// retryingRoundTripper retries requests that fail transiently (HTTP
// 429/502/503/504, or a transport-level error such as a timeout or
// connection reset) with exponential backoff. MCP tool calls are simple,
// idempotent JSON-RPC POSTs, so buffering and replaying the request body on
// retry is safe.
type retryingRoundTripper struct {
	next       http.RoundTripper
	maxRetries int
	baseDelay  time.Duration
}

func (rt *retryingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
	}

	backoff := rt.baseDelay
	var resp *http.Response
	var err error

	for attempt := 0; ; attempt++ {
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err = rt.next.RoundTrip(req)

		retryable := err != nil || isRetryableStatus(resp.StatusCode)
		if !retryable || attempt >= rt.maxRetries {
			return resp, err
		}

		if resp != nil {
			resp.Body.Close()
		}

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}
}

func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
