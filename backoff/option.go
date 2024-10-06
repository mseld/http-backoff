package backoff

import (
	"net/http"
	"time"
)

type (
	RequestLogFunc  func(r *http.Request, err error, attempt int, next time.Duration)
	ResponseLogFunc func(r *http.Request, w *http.Response, attempt int, duration time.Duration)
	ErrorLogFunc    func(r *http.Request, err error, attempt int, duration time.Duration)
)

type config struct {
	// Service name
	service string

	// max number of maxRetry
	maxRetry uint64

	// Minimum time to wait before retrying a request.
	initialInterval time.Duration

	// Maximum time to wait before retrying a request.
	maxInterval time.Duration

	// Exponential backoff multiplier.
	multiplier float64

	// Request timeout.
	timeout *time.Duration

	// client Internal HTTP client.
	client *http.Client

	// RequestLogHook allows a user-supplied function to be called before each retry.
	RequestLogHook RequestLogFunc

	// ResponseLogHook allows a user-supplied function to be called with the response from each HTTP request executed.
	ResponseLogHook ResponseLogFunc

	// ErrorLogHook allows a user-supplied function to be called when an error occurs.
	ErrorLogHook ErrorLogFunc
}

// Option defines a functional option pattern for configuring Config.
type Option interface {
	apply(c *config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithClient sets the HTTP client in Config.
func WithClient(client *http.Client) Option {
	return optionFunc(func(c *config) {
		c.client = client
	})
}

// WithService sets the service name in Config.
func WithService(service string) Option {
	return optionFunc(func(cfg *config) {
		cfg.service = service
	})
}

// WithTimeout sets the request timeout in Config.
func WithTimeout(timeout time.Duration) Option {
	return optionFunc(func(cfg *config) {
		cfg.timeout = &timeout
	})
}

// WithMaxRetry sets the max retry count in Config.
func WithMaxRetry(max uint64) Option {
	return optionFunc(func(c *config) {
		c.maxRetry = max
	})
}

// WithInitialInterval sets the initial retry delay in Config.
func WithInitialInterval(min time.Duration) Option {
	return optionFunc(func(c *config) {
		c.initialInterval = min
	})
}

// WithMaxInterval sets the maximum retry delay in Config.
func WithMaxInterval(max time.Duration) Option {
	return optionFunc(func(c *config) {
		c.maxInterval = max
	})
}

func WithMultiplier(multiplier float64) Option {
	return optionFunc(func(c *config) {
		c.multiplier = multiplier
	})
}

// WithRequestLogHook sets the request log hook in Config.
func WithRequestLogHook(hook RequestLogFunc) Option {
	return optionFunc(func(c *config) {
		c.RequestLogHook = hook
	})
}

// WithResponseLogHook sets the response log hook in Config.
func WithResponseLogHook(hook ResponseLogFunc) Option {
	return optionFunc(func(c *config) {
		c.ResponseLogHook = hook
	})
}

// WithErrorLogHook sets the error hook in Config.
func WithErrorLogHook(hook ErrorLogFunc) Option {
	return optionFunc(func(c *config) {
		c.ErrorLogHook = hook
	})
}
