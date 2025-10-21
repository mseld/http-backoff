package backoff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/oauth2"
)

const (
	DefaultMaxRetry        = 0
	DefaultInitialInterval = 100 * time.Millisecond
	DefaultMultiplier      = 1.5
	DefaultMaxInterval     = 5 * time.Second
	DefaultMaxElapsedTime  = 29 * time.Second
)

// RetryableSet is a set of HTTP status codes (4xx) that are retryable.
var RetryableSet = map[int]struct{}{
	http.StatusRequestTimeout:  {},
	http.StatusTooEarly:        {},
	http.StatusTooManyRequests: {},
}

// Response represents an HTTP response.
type Response struct {
	Status     string      `json:"status,omitempty"`
	StatusCode int         `json:"status_code,omitempty"`
	Header     http.Header `json:"header,omitempty"`
	Body       []byte      `json:"body,omitempty"`
}

// RetryableError is an error that can be retried.
type RetryableError struct {
	Response *http.Response `json:"-"`
	Err      error          `json:"error,omitempty"`
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Message() string {
	if e.Response != nil {
		return fmt.Sprintf("http-client: failed to %s %s response: %s", e.Response.Request.Method, e.Response.Request.URL, e.Response.Status)
	}

	return e.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func (e *RetryableError) Is(target error) bool {
	_, ok := target.(*RetryableError)
	return ok
}

type BackoffClient struct {
	*http.Client
	cfg             config
	backOffStrategy backoff.BackOff
}

func NewBackoffClient(opts ...Option) *BackoffClient {
	cfg := config{
		agentName:       "http-backoff-client",
		maxRetry:        DefaultMaxRetry,
		initialInterval: DefaultInitialInterval,
		maxInterval:     DefaultMaxInterval,
		multiplier:      DefaultMultiplier,
		maxElapsedTime:  DefaultMaxElapsedTime,
		client:          http.DefaultClient,
		RequestLogHook:  func(r *http.Request, err error, n int, next time.Duration) {},
		ResponseLogHook: func(r *http.Request, w *http.Response, n int, d time.Duration) {},
		ErrorLogHook:    func(r *http.Request, err error, n int, d time.Duration) {},
	}

	for _, opt := range opts {
		opt.apply(&cfg)
	}

	// Create exponential backoff with configured parameters
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = cfg.initialInterval
	expBackoff.MaxInterval = cfg.maxInterval
	expBackoff.Multiplier = cfg.multiplier
	expBackoff.MaxElapsedTime = cfg.maxElapsedTime

	var backOffStrategy backoff.BackOff = expBackoff
	if cfg.maxRetry > 0 {
		backOffStrategy = backoff.WithMaxRetries(expBackoff, cfg.maxRetry)
	}

	return &BackoffClient{
		cfg:             cfg,
		backOffStrategy: backOffStrategy,
		Client:          cfg.client,
	}
}

// Get performs an HTTP GET request.
func (c *BackoffClient) Get(ctx context.Context, url string, headers map[string]string) (*Response, error) {
	req, err := NewRequestBuilder().
		Method(http.MethodGet).
		URL(url).
		Headers(headers).
		Build(ctx)
	if err != nil {
		return nil, err
	}

	return c.Execute(req)
}

func (c *BackoffClient) Post(ctx context.Context, url string, body io.Reader, headers map[string]string) (*Response, error) {
	req, err := NewRequestBuilder().
		Method(http.MethodPost).
		URL(url).
		Body(body).
		Headers(headers).
		Build(ctx)
	if err != nil {
		return nil, err
	}

	return c.Execute(req)
}

// PostJSON performs an HTTP POST request with a JSON body.
func (c *BackoffClient) PostJSON(ctx context.Context, url string, body any, headers map[string]string) (*Response, error) {
	req, err := NewRequestBuilder().
		Method(http.MethodPost).
		URL(url).
		BodyJSON(body).
		Headers(headers).
		Build(ctx)
	if err != nil {
		return nil, err
	}

	return c.Execute(req)
}

// PostForm performs an HTTP POST request with form data.
func (c *BackoffClient) PostForm(ctx context.Context, url string, form map[string]string, headers map[string]string) (*Response, error) {
	req, err := NewRequestBuilder().
		Method(http.MethodPost).
		URL(url).
		PostForm(form).
		Headers(headers).
		Build(ctx)
	if err != nil {
		return nil, err
	}

	return c.Execute(req)
}

// Put performs an HTTP PUT request.
func (c *BackoffClient) Put(ctx context.Context, url string, body io.Reader, headers map[string]string) (*Response, error) {
	req, err := NewRequestBuilder().
		Method(http.MethodPut).
		URL(url).
		Body(body).
		Headers(headers).
		Build(ctx)
	if err != nil {
		return nil, err
	}

	return c.Execute(req)
}

// PutJSON performs an HTTP PUT request with a JSON body.
func (c *BackoffClient) PutJSON(ctx context.Context, url string, body any, headers map[string]string) (*Response, error) {
	req, err := NewRequestBuilder().
		Method(http.MethodPut).
		URL(url).
		BodyJSON(body).
		Headers(headers).
		Build(ctx)
	if err != nil {
		return nil, err
	}

	return c.Execute(req)
}

// Patch performs an HTTP PATCH request.
func (c *BackoffClient) Patch(ctx context.Context, url string, body io.Reader, headers map[string]string) (*Response, error) {
	req, err := NewRequestBuilder().
		Method(http.MethodPatch).
		URL(url).
		Body(body).
		Headers(headers).
		Build(ctx)
	if err != nil {
		return nil, err
	}

	return c.Execute(req)
}

// PatchJSON performs an HTTP PATCH request with a JSON body.
func (c *BackoffClient) PatchJSON(ctx context.Context, url string, body any, headers map[string]string) (*Response, error) {
	req, err := NewRequestBuilder().
		Method(http.MethodPatch).
		URL(url).
		BodyJSON(body).
		Headers(headers).
		Build(ctx)
	if err != nil {
		return nil, err
	}

	return c.Execute(req)
}

// Delete performs an HTTP DELETE request.
func (c *BackoffClient) Delete(ctx context.Context, url string, headers map[string]string) (*Response, error) {
	req, err := NewRequestBuilder().
		Method(http.MethodDelete).
		URL(url).
		Headers(headers).
		Build(ctx)
	if err != nil {
		return nil, err
	}

	return c.Execute(req)
}

// Execute performs the HTTP request with retry logic.
func (c *BackoffClient) Execute(r *http.Request) (*Response, error) {
	attempt := 0
	f := func() (*Response, error) {
		attempt++
		startTime := time.Now()

		// Clone the request for each retry to preserve the original
		clonedReq := cloneRequest(r)

		resp, err := c.execute(clonedReq)
		if err != nil {
			c.cfg.ErrorLogHook(clonedReq, err, attempt, time.Since(startTime))

			var retryableErr *RetryableError
			if errors.As(err, &retryableErr) {
				return nil, err
			}

			return nil, backoff.Permanent(err)
		}

		c.cfg.ResponseLogHook(clonedReq, resp, attempt, time.Since(startTime))

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, backoff.Permanent(fmt.Errorf("failed to read response body: %w", err))
		}

		return &Response{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Body:       body,
		}, nil
	}

	notify := func(err error, next time.Duration) {
		c.cfg.RequestLogHook(r, err, attempt, next)
	}

	// Reset backoff before each Execute call to ensure consistent behavior
	c.backOffStrategy.Reset()

	return backoff.RetryNotifyWithData(f, c.backOffStrategy, notify)
}

// execute performs the actual HTTP request.
func (c *BackoffClient) execute(r *http.Request) (*http.Response, error) {
	if c.cfg.timeout != nil {
		ctx, cancel := context.WithTimeout(r.Context(), *c.cfg.timeout)
		defer cancel()
		r = r.WithContext(ctx)
	}

	if c.cfg.agentName != "" {
		r.Header.Set("User-Agent", c.cfg.agentName)
	}

	resp, err := c.Do(r)
	if err != nil {
		if ErrorRetryPolicy(err) {
			return nil, &RetryableError{Err: err}
		}
		return nil, err
	}

	if err := ResponseRetryPolicy(resp); err != nil {
		// Close the body on retryable errors to prevent connection leaks
		resp.Body.Close()
		return nil, &RetryableError{
			Response: resp,
			Err:      err,
		}
	}

	return resp, nil
}

// Close closes idle connections. Call this when you're done with the client
// or want to clean up connections (e.g., in defer statements or cleanup code).
func (c *BackoffClient) Close() {
	c.CloseIdleConnections()
}

// cloneRequest creates a copy of the request with a fresh body if needed.
func cloneRequest(r *http.Request) *http.Request {
	cloned := r.Clone(r.Context())

	// If the original request has a GetBody function, use it to restore the body
	if r.Body != nil && r.GetBody != nil {
		body, err := r.GetBody()
		if err == nil {
			cloned.Body = body
		}
	}

	return cloned
}

func ErrorRetryPolicy(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Retry on OAuth2 token retrieval errors
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) {
		// Don't retry on client errors (4xx) except for specific retryable ones
		if retrieveErr.Response != nil {
			statusCode := retrieveErr.Response.StatusCode
			if statusCode >= 400 && statusCode < 500 {
				_, retryable := RetryableSet[statusCode]
				return retryable
			}
			// Retry on 5xx server errors
			return statusCode >= 500
		}
		return true
	}

	return false
}

func ResponseRetryPolicy(resp *http.Response) error {
	// Check retryable 4xx status codes
	if _, ok := RetryableSet[resp.StatusCode]; ok {
		return fmt.Errorf("retryable status code: %s", resp.Status)
	}

	// Retry on 5xx errors except 501 Not Implemented
	if resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented {
		return fmt.Errorf("server error: %s", resp.Status)
	}

	return nil
}

func Unmarshal[T any](response []byte) (T, error) {
	var result T
	err := json.Unmarshal(response, &result)
	return result, err
}
