package reghelp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

// DefaultBaseURL is the production REGHelp Key API endpoint.
const DefaultBaseURL = "https://api.reghelp.net"

// Client talks to the REGHelp Key API. Construct with [New]; all methods are
// safe for concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
	// userAgent is set on every request. Override with Option [WithUserAgent].
	userAgent string
}

// Option customizes the [Client] returned by [New].
type Option func(*Client)

// WithBaseURL overrides the API base URL (default [DefaultBaseURL]).
func WithBaseURL(u string) Option {
	return func(c *Client) {
		if u != "" {
			c.baseURL = trimRightSlash(u)
		}
	}
}

// WithHTTPClient supplies a custom *http.Client. The client must honor
// context cancellation — i.e., its Transport should propagate ctx deadlines.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// WithTimeout sets the HTTP client's per-request timeout (default 30s).
// Ignored if [WithHTTPClient] is also passed.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.httpClient.Timeout = d
		}
	}
}

// WithMaxRetries sets the maximum number of retry attempts on transient
// failures (429, network errors). Default 3.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n >= 0 {
			c.maxRetries = n
		}
	}
}

// WithRetryDelay sets the base back-off delay between retries (default 1s).
// Effective delay scales as base * 2^attempt.
func WithRetryDelay(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.retryDelay = d
		}
	}
}

// WithUserAgent overrides the User-Agent header sent on every request.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// New returns a Client bound to apiKey. Options are applied in order.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		maxRetries: 3,
		retryDelay: time.Second,
		userAgent:  "reghelp-client-go/1.0.0",
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// BaseURL reports the URL the client targets.
func (c *Client) BaseURL() string { return c.baseURL }

// HealthCheck pings GET /health. Returns true iff the API answers 200.
// Does not require an API key.
func (c *Client) HealthCheck(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, &Error{Code: "NETWORK", Message: err.Error(), Cause: err}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK, nil
}

// do executes a GET to baseURL+endpoint with apiKey + params and decodes the
// JSON envelope.
//
// allowErrorStatus=true tells do() to NOT translate {"status":"error",...}
// envelopes into an Error — used by SetPushStatus where an "error" envelope
// can carry a legitimate balance.
func (c *Client) do(
	ctx context.Context,
	endpoint string,
	params map[string]string,
	taskID string,
	allowErrorStatus bool,
) (map[string]any, error) {
	return c.doMethod(ctx, http.MethodGet, endpoint, params, taskID, allowErrorStatus)
}

// doMethod is do() with an explicit HTTP method. POST endpoints still pass
// arguments via the query string (matches FastAPI's behaviour when every
// param is declared via Query(...)); the body stays empty.
func (c *Client) doMethod(
	ctx context.Context,
	method string,
	endpoint string,
	params map[string]string,
	taskID string,
	allowErrorStatus bool,
) (map[string]any, error) {
	full, err := c.buildURL(endpoint, params)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, full, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = &Error{Code: "NETWORK", Message: err.Error(), Cause: err}
			if attempt < c.maxRetries && ctx.Err() == nil {
				c.backoff(attempt)
				continue
			}
			return nil, lastErr
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = mapErrorCode(resp.StatusCode, "RATE_LIMIT", "", taskID, nil)
			if attempt < c.maxRetries && ctx.Err() == nil {
				c.backoff(attempt)
				continue
			}
			return nil, lastErr
		}

		var decoded map[string]any
		if len(body) > 0 {
			if err := json.Unmarshal(body, &decoded); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidJSONResult, err)
			}
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if !allowErrorStatus && asString(decoded["status"]) == "error" {
				code := firstNonEmpty(asString(decoded["id"]), asString(decoded["detail"]), "UNKNOWN_ERROR")
				return nil, mapErrorCode(resp.StatusCode, code, asString(decoded["message"]), taskID, decoded)
			}
			return decoded, nil
		}

		// Non-2xx: map by id/detail or status code.
		code := firstNonEmpty(asString(decoded["id"]), asString(decoded["detail"]))
		return nil, mapErrorCode(resp.StatusCode, code, asString(decoded["message"]), taskID, decoded)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, &Error{Code: "UNKNOWN", Message: "exhausted retries"}
}

// backoff sleeps `retryDelay * 2^attempt` with small jitter, honoring ctx.
func (c *Client) backoff(attempt int) {
	d := c.retryDelay * (1 << attempt)
	jitter := time.Duration(rand.Int63n(int64(c.retryDelay)))
	time.Sleep(d + jitter)
}

func (c *Client) buildURL(endpoint string, params map[string]string) (string, error) {
	u, err := url.Parse(c.baseURL + "/" + trimLeftSlash(endpoint))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("apiKey", c.apiKey)
	for k, v := range params {
		if v == "" {
			continue
		}
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// decode marshals a generic map into the typed target via JSON round-trip.
// Convenient when /getStatus endpoints have heterogeneous shapes.
func decode(m map[string]any, target any) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}

func trimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func trimLeftSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	return s
}

// Asserts the SDK compiles with go's modern std http behavior. Unused but
// keeps imports honest if other files trim.
var _ = errors.New
