package backoff

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// ContentTypeJSON is the value for the Content-Type header for JSON.
	ContentTypeJSON = "application/json"
	// ContentTypeForm is the value for the Content-Type header for form data.
	ContentTypeForm = "application/x-www-form-urlencoded"
	// ContentTypeText is the value for the Content-Type header for plain text.
	ContentTypeText = "text/plain"
	// ContentTypeHTML is the value for the Content-Type header for HTML.
	ContentTypeHTML = "text/html"
	// ContentTypeXML is the value for the Content-Type header for XML.
	ContentTypeXML = "application/xml"
	// ContentTypeOctetStream is the value for the Content-Type header for binary data.
	ContentTypeOctetStream = "application/octet-stream"
)

var (
	// UserAgentHeader is the key for the User-Agent header.
	UserAgentHeader = http.CanonicalHeaderKey("User-Agent")
	// ContentTypeHeader is the key for the Content-Type header.
	ContentTypeHeader = http.CanonicalHeaderKey("Content-Type")
)

type RequestBuilder struct {
	method    string
	url       string
	query     url.Values
	headers   map[string]string
	form      url.Values
	body      io.Reader
	bodyBytes []byte // Store body bytes for retry support
}

// NewRequestBuilder creates a new RequestBuilder instance.
func NewRequestBuilder() *RequestBuilder {
	return &RequestBuilder{
		query:   url.Values{},
		headers: make(map[string]string),
	}
}

// Method sets the request method.
func (rb *RequestBuilder) Method(method string) *RequestBuilder {
	rb.method = method
	return rb
}

// URL sets the request URL.
func (rb *RequestBuilder) URL(url string) *RequestBuilder {
	rb.url = url
	return rb
}

// QueryParam adds a query parameter to the URL.
func (rb *RequestBuilder) QueryParam(key, value string) *RequestBuilder {
	rb.query.Add(key, value)
	return rb
}

// Query adds multiple query parameters to the URL.
func (rb *RequestBuilder) Query(query url.Values) *RequestBuilder {
	if len(query) == 0 {
		return rb
	}

	if len(rb.query) == 0 {
		rb.query = query
		return rb
	}

	for key, values := range query {
		for _, value := range values {
			rb.query.Add(key, value)
		}
	}
	return rb
}

// Header sets a header.
func (rb *RequestBuilder) Header(key, value string) *RequestBuilder {
	rb.headers[key] = value
	return rb
}

// Headers sets multiple headers.
func (rb *RequestBuilder) Headers(headers map[string]string) *RequestBuilder {
	for key, value := range headers {
		rb.headers[key] = value
	}
	return rb
}

// ContentType sets the Content-Type header.
func (rb *RequestBuilder) ContentType(contentType string) *RequestBuilder {
	rb.headers[ContentTypeHeader] = contentType
	return rb
}

// UserAgent sets the User-Agent header.
func (rb *RequestBuilder) UserAgent(userAgent string) *RequestBuilder {
	rb.headers[UserAgentHeader] = userAgent
	return rb
}

// Body sets the body of the request.
func (rb *RequestBuilder) Body(body io.Reader) *RequestBuilder {
	rb.body = body
	// Try to read body into bytes for retry support
	if body != nil {
		if data, err := io.ReadAll(body); err == nil {
			rb.bodyBytes = data
		}
	}
	return rb
}

// BodyString sets the body from a string.
func (rb *RequestBuilder) BodyString(body string) *RequestBuilder {
	rb.bodyBytes = []byte(body)
	rb.body = strings.NewReader(body)
	return rb
}

// BodyBytes sets the body from a byte slice.
func (rb *RequestBuilder) BodyBytes(body []byte) *RequestBuilder {
	rb.bodyBytes = body
	rb.body = bytes.NewReader(body)
	return rb
}

// BodyJSON sets the body from a JSON object and sets Content-Type to application/json.
func (rb *RequestBuilder) BodyJSON(data any) *RequestBuilder {
	buffer, err := json.Marshal(data)
	if err != nil {
		return rb
	}
	rb.bodyBytes = buffer
	rb.body = bytes.NewBuffer(buffer)
	rb.headers[ContentTypeHeader] = ContentTypeJSON
	return rb
}

// PostForm sets the form data for a POST request.
func (rb *RequestBuilder) PostForm(form map[string]string) *RequestBuilder {
	for key, value := range form {
		rb.form.Add(key, value)
	}
	return rb
}

// Build creates and returns the final *http.Request.
func (rb *RequestBuilder) Build(ctx context.Context) (*http.Request, error) {
	u, err := url.Parse(rb.url)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Add query parameters to the URL if any
	if len(rb.query) > 0 {
		u.RawQuery = rb.query.Encode()
	}

	var bodyReader io.Reader

	// Encode form data if present
	if len(rb.form) > 0 {
		if rb.method == "" {
			rb.method = http.MethodPost
		}
		rb.headers[ContentTypeHeader] = ContentTypeForm
		formData := rb.form.Encode()
		rb.bodyBytes = []byte(formData)
		bodyReader = strings.NewReader(formData)
	} else if len(rb.bodyBytes) > 0 {
		bodyReader = bytes.NewReader(rb.bodyBytes)
	} else {
		bodyReader = rb.body
	}

	// Default to GET if no method specified
	if rb.method == "" {
		rb.method = http.MethodGet
	}

	// Create the request
	r, err := http.NewRequestWithContext(ctx, rb.method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set GetBody for retry support if we have bodyBytes
	if len(rb.bodyBytes) > 0 {
		r.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(rb.bodyBytes)), nil
		}
	}

	// Add headers to the request
	for key, value := range rb.headers {
		r.Header.Set(key, value)
	}

	return r, nil
}
