package tracing

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

func Test_safeFullURL(t *testing.T) {
	testCases := []struct {
		desc            string
		safeQueryParams []string
		originalURL     *url.URL
		expectedURL     *url.URL
	}{
		{
			desc:        "Nil URL",
			originalURL: nil,
			expectedURL: nil,
		},
		{
			desc:        "No query parameters",
			originalURL: &url.URL{Scheme: "https", Host: "example.com"},
			expectedURL: &url.URL{Scheme: "https", Host: "example.com"},
		},
		{
			desc:        "All query parameters redacted",
			originalURL: &url.URL{Scheme: "https", Host: "example.com", RawQuery: "foo=bar&baz=qux"},
			expectedURL: &url.URL{Scheme: "https", Host: "example.com", RawQuery: "baz=REDACTED&foo=REDACTED"},
		},
		{
			desc:            "Some query parameters unredacted",
			safeQueryParams: []string{"foo"},
			originalURL:     &url.URL{Scheme: "https", Host: "example.com", RawQuery: "foo=bar&baz=qux"},
			expectedURL:     &url.URL{Scheme: "https", Host: "example.com", RawQuery: "baz=REDACTED&foo=bar"},
		},
		{
			desc:            "User info and some query parameters redacted",
			safeQueryParams: []string{"foo"},
			originalURL:     &url.URL{Scheme: "https", Host: "example.com", User: url.UserPassword("username", "password"), RawQuery: "foo=bar&baz=qux"},
			expectedURL:     &url.URL{Scheme: "https", Host: "example.com", User: url.UserPassword("REDACTED", "REDACTED"), RawQuery: "baz=REDACTED&foo=bar"},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			tr := NewTracer(nil, nil, nil, test.safeQueryParams)

			gotURL := tr.safeURL(test.originalURL)

			assert.Equal(t, test.expectedURL, gotURL)
		})
	}
}

// TestNewTracer_HeadersCanonicalization tests that NewTracer properly canonicalizes headers.
func TestNewTracer_HeadersCanonicalization(t *testing.T) {
	testCases := []struct {
		desc                     string
		inputHeaders             []string
		expectedCanonicalHeaders []string
	}{
		{
			desc:                     "Empty headers",
			inputHeaders:             []string{},
			expectedCanonicalHeaders: []string{},
		},
		{
			desc:                     "Already canonical headers",
			inputHeaders:             []string{"Content-Type", "User-Agent", "Accept-Encoding"},
			expectedCanonicalHeaders: []string{"Content-Type", "User-Agent", "Accept-Encoding"},
		},
		{
			desc:                     "Lowercase headers",
			inputHeaders:             []string{"content-type", "user-agent", "accept-encoding"},
			expectedCanonicalHeaders: []string{"Content-Type", "User-Agent", "Accept-Encoding"},
		},
		{
			desc:                     "Mixed case headers",
			inputHeaders:             []string{"CoNtEnT-tYpE", "uSeR-aGeNt", "aCcEpT-eNcOdInG"},
			expectedCanonicalHeaders: []string{"Content-Type", "User-Agent", "Accept-Encoding"},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			// Create a mock tracer using a no-op tracer from OpenTelemetry
			mockTracer := noop.NewTracerProvider().Tracer("test")

			// Test capturedRequestHeaders
			tracer := NewTracer(mockTracer, test.inputHeaders, nil, nil)
			assert.Equal(t, test.expectedCanonicalHeaders, tracer.capturedRequestHeaders)
			assert.Nil(t, tracer.capturedResponseHeaders)

			// Test capturedResponseHeaders
			tracer = NewTracer(mockTracer, nil, test.inputHeaders, nil)
			assert.Equal(t, test.expectedCanonicalHeaders, tracer.capturedResponseHeaders)
			assert.Nil(t, tracer.capturedRequestHeaders)
		})
	}
}
