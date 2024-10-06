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
	DefaultMaxElapsedTime  = 30 * time.Minute
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
		service:         "http-client",
		maxRetry:        DefaultMaxRetry,
		initialInterval: DefaultInitialInterval,
		maxInterval:     DefaultMaxInterval,
		multiplier:      DefaultMultiplier,
		client:          NewDefaultClient(),
		RequestLogHook:  func(r *http.Request, err error, n int, next time.Duration) {},
		ResponseLogHook: func(r *http.Request, w *http.Response, n int, d time.Duration) {},
		ErrorLogHook:    func(r *http.Request, err error, n int, d time.Duration) {},
	}

	for _, opt := range opts {
		opt.apply(&cfg)
	}

	var backOffStrategy backoff.BackOff = backoff.NewExponentialBackOff()
	if cfg.maxRetry > 0 {
		backOffStrategy = backoff.WithMaxRetries(backoff.NewExponentialBackOff(), cfg.maxRetry)
	}

	return &BackoffClient{
		cfg:             cfg,
		backOffStrategy: backOffStrategy,
		Client:          http.DefaultClient,
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

// Post performs an HTTP POST request with a JSON body.
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

// PatchJSON performs an HTTP PATCH request.
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

// Execute performs the HTTP request and handles response.
func (c *BackoffClient) Execute(r *http.Request) (*Response, error) {
	attempt := 0
	f := func() (*Response, error) {
		attempt++
		startTime := time.Now()
		resp, err := c.execute(r)
		if err != nil {
			c.cfg.ErrorLogHook(r, err, attempt, time.Since(startTime))

			if errors.Is(err, &RetryableError{}) {
				return nil, err
			}

			return nil, backoff.Permanent(err)
		}

		c.cfg.ResponseLogHook(r, resp, attempt, time.Since(startTime))

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			// c.cfg.ErrorHook(r, err, attempt, time.Since(startTime))
			return nil, err
		}

		defer resp.Body.Close()

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

	return backoff.RetryNotifyWithData(f, c.backOffStrategy, notify)
}

// execute performs the HTTP request and handles response.
func (c *BackoffClient) execute(r *http.Request) (*http.Response, error) {
	defer c.CloseIdleConnections()

	if c.cfg.timeout != nil {
		ctx, cancel := context.WithTimeout(r.Context(), *c.cfg.timeout)
		r = r.WithContext(ctx)
		defer cancel()
	}

	resp, err := c.Do(r)
	if err != nil {
		if ErrorRetryPolicy(err) {
			return nil, &RetryableError{
				Err: err,
			}
		}

		return nil, err
	}

	if err := ResponseRetryPolicy(resp); err != nil {
		return nil, &RetryableError{
			Response: resp,
			Err:      err,
		}
	}

	return resp, nil
}

func ErrorRetryPolicy(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// retry on oauth2 errors
	if errors.Is(err, &oauth2.RetrieveError{}) {
		return true
	}

	return false
}

func ResponseRetryPolicy(resp *http.Response) error {
	// RetryableSet is recoverable status codes.
	if _, ok := RetryableSet[resp.StatusCode]; ok {
		return fmt.Errorf("status code retryable: %s", resp.Status)
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like [InternalServerError, BadGateway, ServiceUnavailable, GatewayTimeout).
	if resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented {
		return fmt.Errorf("unexpected status code %s", resp.Status)
	}

	return nil
}

func Unmarshal[T any](response []byte) (T, error) {
	var result T
	err := json.Unmarshal(response, &result)
	return result, err
}
