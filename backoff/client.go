package backoff

import (
	"context"
	"net/http"
	"net/http/httptrace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

// WithAgentNameAttribute sets
func WithAgentNameAttribute(name string) attribute.KeyValue {
	return attribute.String("agent-name", name)
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
