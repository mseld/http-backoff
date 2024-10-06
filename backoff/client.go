package backoff

import (
	"context"
	"net"
	"net/http"
	"net/http/httptrace"
	"runtime"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// NewClient creates an HTTP client
func NewDefaultClient() *http.Client {
	return &http.Client{}
}

// NewClientWithOtel creates an HTTP client with OpenTelemetry instrumentation.
func NewDefaultClientWithOtel(attributes ...attribute.KeyValue) *http.Client {
	return newClientWithTransport(http.DefaultTransport, attributes...)
}

// NewOAuth2Client creates an HTTP client using OAuth2 credentials.
func NewOAuth2Client(credentials clientcredentials.Config) *http.Client {
	return credentials.Client(context.Background())
}

// NewOAuth2ClientWithOtel Creates an HTTP client using OAuth2 credentials with OpenTelemetry instrumentation.
func NewOAuth2ClientWithOtel(credentials clientcredentials.Config, attributes ...attribute.KeyValue) *http.Client {
	client := credentials.Client(context.Background())
	client.Transport = newInstrumentedTransport(client.Transport, attributes...)
	return client
}

// NewPooledClient returns a new http.Client with similar default values to
// http.Client, but with a shared Transport. Do not use this function for
// transient clients as it can leak file descriptors over time. Only use this
// for clients that will be re-used for the same host(s).
func NewPooledClient() *http.Client {
	return &http.Client{
		Transport: NewPooledTransport(),
	}
}

// NewPooledClientWithOtel returns a new http.Client with similar default values to
// http.Client, but with a shared Transport and OpenTelemetry instrumentation.
// Do not use this function for transient clients as it can leak file descriptors
// over time. Only use this for clients that will be re-used for the same host(s).
func NewPooledClientWithOtel(attributes ...attribute.KeyValue) *http.Client {
	return newClientWithTransport(NewPooledTransport(), attributes...)
}

// PooledOAuth2ClientWithOtel combines a pooled HTTP client with OAuth2 authentication and OpenTelemetry instrumentation.
func NewPooledOAuth2ClientWithOtel(credentials clientcredentials.Config, attributes ...attribute.KeyValue) *http.Client {
	ctx := context.Background()
	client := credentials.Client(ctx)
	transporter := &oauth2.Transport{
		Source: credentials.TokenSource(ctx),
		Base:   NewPooledTransport(),
	}
	client.Transport = newInstrumentedTransport(transporter, attributes...)
	return client
}

// NewPooledTransport returns a new http.Transport with similar default
// values to http.DefaultTransport. Do not use this for transient transports as
// it can leak file descriptors over time. Only use this for transports that
// will be re-used for the same host(s).
func NewPooledTransport() *http.Transport {
	const defaultMaxIdleConns = 100
	const defaultTimeout = 30 * time.Second
	const defaultKeepAlive = 30 * time.Second
	const defaultIdleConnTimeout = 90 * time.Second
	const defaultTLSHandshakeTimeout = 10 * time.Second
	const defaultExpectContinueTimeout = 1 * time.Second

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   defaultTimeout,
			KeepAlive: defaultKeepAlive,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          defaultMaxIdleConns,
		IdleConnTimeout:       defaultIdleConnTimeout,
		TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
		ExpectContinueTimeout: defaultExpectContinueTimeout,
		ForceAttemptHTTP2:     true,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}
	return transport
}

// WithServiceAttribute sets
func WithServiceAttribute(service string) attribute.KeyValue {
	return attribute.String("service", service)
}

// newClientWithTransport creates a new HTTP client with a given transport and optional OpenTelemetry instrumentation.
func newClientWithTransport(transport http.RoundTripper, attributes ...attribute.KeyValue) *http.Client {
	return &http.Client{
		Transport: newInstrumentedTransport(transport, attributes...),
	}
}

// newInstrumentedTransport adds OpenTelemetry instrumentation to a given transport.
func newInstrumentedTransport(transport http.RoundTripper, attributes ...attribute.KeyValue) http.RoundTripper {
	opts := withOtelOptions(attributes...)
	return otelhttp.NewTransport(transport, opts...)
}

func withOtelOptions(attributes ...attribute.KeyValue) []otelhttp.Option {
	opts := []otelhttp.Option{
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
			return otelhttptrace.NewClientTrace(ctx)
		}),
	}

	if len(attributes) > 0 {
		opts = append(opts, otelhttp.WithSpanOptions(trace.WithAttributes(attributes...)))
	}

	return opts
}
