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
	// agentName sets the User-Agent header
	agentName string

	// maxRetry is the maximum number of retry attempts (0 = unlimited with MaxElapsedTime limit)
	maxRetry uint64

	// initialInterval is the minimum time to wait before retrying a request
	initialInterval time.Duration

	// maxInterval is the maximum time to wait before retrying a request
	maxInterval time.Duration

	// maxElapsedTime is the maximum total time for all retries
	maxElapsedTime time.Duration

	// multiplier is the exponential backoff multiplier
	multiplier float64

	// timeout is the request timeout per attempt
	timeout *time.Duration

	// client is the internal HTTP client
	client *http.Client

	// RequestLogHook is called before each retry
	RequestLogHook RequestLogFunc

	// ResponseLogHook is called with the response from each HTTP request
	ResponseLogHook ResponseLogFunc

	// ErrorLogHook is called when an error occurs
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
		if client != nil {
			c.client = client
		}
	})
}

// WithAgentName sets the agent name in Config.
func WithAgentName(name string) Option {
	return optionFunc(func(cfg *config) {
		cfg.agentName = name
	})
}

// WithTimeout sets the request timeout per attempt in Config.
func WithTimeout(timeout time.Duration) Option {
	return optionFunc(func(cfg *config) {
		cfg.timeout = &timeout
	})
}

// WithMaxRetry sets the max retry count in Config.
// Use 0 for unlimited retries (limited by MaxElapsedTime).
func WithMaxRetry(max uint64) Option {
	return optionFunc(func(c *config) {
		c.maxRetry = max
	})
}

// WithInitialInterval sets the initial retry delay in Config.
func WithInitialInterval(interval time.Duration) Option {
	return optionFunc(func(c *config) {
		c.initialInterval = interval
	})
}

// WithMaxInterval sets the maximum retry delay in Config.
func WithMaxInterval(interval time.Duration) Option {
	return optionFunc(func(c *config) {
		c.maxInterval = interval
	})
}

// WithMaxElapsedTime sets the maximum total time for all retries.
func WithMaxElapsedTime(duration time.Duration) Option {
	return optionFunc(func(c *config) {
		c.maxElapsedTime = duration
	})
}

// WithMultiplier sets the exponential backoff multiplier.
func WithMultiplier(multiplier float64) Option {
	return optionFunc(func(c *config) {
		c.multiplier = multiplier
	})
}

// WithRequestLogHook sets the request log hook in Config.
func WithRequestLogHook(hook RequestLogFunc) Option {
	return optionFunc(func(c *config) {
		if hook != nil {
			c.RequestLogHook = hook
		}
	})
}

// WithResponseLogHook sets the response log hook in Config.
func WithResponseLogHook(hook ResponseLogFunc) Option {
	return optionFunc(func(c *config) {
		if hook != nil {
			c.ResponseLogHook = hook
		}
	})
}

// WithErrorLogHook sets the error hook in Config.
func WithErrorLogHook(hook ErrorLogFunc) Option {
	return optionFunc(func(c *config) {
		if hook != nil {
			c.ErrorLogHook = hook
		}
	})
}
