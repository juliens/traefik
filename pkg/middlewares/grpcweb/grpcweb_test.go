package grpcweb

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

func TestMiddleware_ServeHTTP_NonGrpcWebRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "passed-through")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("regular response"))
	})

	config := dynamic.GrpcWeb{
		AllowOrigins: []string{"*"},
	}
	middleware := New(context.Background(), next, config, "test")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "passed-through", rr.Header().Get("X-Test"))
	assert.Equal(t, "regular response", rr.Body.String())
}

func TestMiddleware_ServeHTTP_GrpcWebRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/grpc", r.Header.Get("Content-Type"))
		assert.Equal(t, int64(-1), r.ContentLength)

		w.Header().Set("Content-Type", "application/grpc")
		w.Header().Set("Grpc-Status", "0")
		w.Header().Set("Grpc-Message", "OK")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("grpc response"))
	})

	config := dynamic.GrpcWeb{
		AllowOrigins: []string{"*"},
	}
	middleware := New(context.Background(), next, config, "test")

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("grpc request"))
	req.Header.Set("Content-Type", "application/grpc-web")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/grpc-web", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Access-Control-Expose-Headers"), "grpc-status")
	assert.Contains(t, rr.Header().Get("Access-Control-Expose-Headers"), "grpc-message")
}

func TestMiddleware_ServeHTTP_GrpcWebTextRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/grpc", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "grpc request", string(body))

		w.Header().Set("Content-Type", "application/grpc")
		w.Header().Set("Grpc-Status", "0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("grpc response"))
	})

	config := dynamic.GrpcWeb{
		AllowOrigins: []string{"*"},
	}
	middleware := New(context.Background(), next, config, "test")

	encodedBody := base64.StdEncoding.EncodeToString([]byte("grpc request"))
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(encodedBody))
	req.Header.Set("Content-Type", "application/grpc-web-text")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/grpc-web-text", rr.Header().Get("Content-Type"))

	decoded, err := base64.StdEncoding.DecodeString(rr.Body.String())
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "grpc response")
}

func TestMiddleware_ServeHTTP_PreflightRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Next handler should not be called for preflight")
	})

	config := dynamic.GrpcWeb{
		AllowOrigins: []string{"https://example.com"},
	}
	middleware := New(context.Background(), next, config, "test")

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "x-grpc-web,content-type")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
}

func TestConvertToGrpcRequest_Binary(t *testing.T) {
	body := strings.NewReader("grpc request body")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/grpc-web+proto")
	req.Header.Set("Content-Length", "17")
	req.Header.Set("X-Custom-Header", "value")

	newReq, isTextFormat := convertToGrpcRequest(req)

	assert.False(t, isTextFormat)
	assert.Equal(t, "application/grpc+proto", newReq.Header.Get("Content-Type"))
	assert.Equal(t, int64(-1), newReq.ContentLength)
	assert.Equal(t, "value", newReq.Header.Get("X-Custom-Header"))
	assert.Equal(t, 2, newReq.ProtoMajor)
	assert.Equal(t, 0, newReq.ProtoMinor)

	bodyBytes, err := io.ReadAll(newReq.Body)
	require.NoError(t, err)
	assert.Equal(t, "grpc request body", string(bodyBytes))
}

func TestConvertToGrpcRequest_Text(t *testing.T) {
	originalBody := "grpc request body"
	encodedBody := base64.StdEncoding.EncodeToString([]byte(originalBody))

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(encodedBody))
	req.Header.Set("Content-Type", "application/grpc-web-text+proto")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(encodedBody)))

	newReq, isTextFormat := convertToGrpcRequest(req)

	assert.True(t, isTextFormat)
	assert.Equal(t, "application/grpc+proto", newReq.Header.Get("Content-Type"))
	assert.Equal(t, int64(-1), newReq.ContentLength)

	bodyBytes, err := io.ReadAll(newReq.Body)
	require.NoError(t, err)
	assert.Equal(t, originalBody, string(bodyBytes))
}

func TestGrpcWebResponse_Headers(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := newGrpcWebResponse(rr, false)

	// Test header access
	resp.Header().Set("X-Test", "value")
	assert.Equal(t, "value", resp.Header().Get("X-Test"))

	// Headers should not be written to underlying response yet
	assert.Empty(t, rr.Header().Get("X-Test"))
}

func TestGrpcWebResponse_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := newGrpcWebResponse(rr, false)

	resp.Header().Set("Content-Type", "application/grpc")
	resp.Header().Set("Grpc-Status", "0")
	resp.Header().Set("Grpc-Message", "OK")

	resp.WriteHeader(http.StatusOK)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/grpc-web", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Access-Control-Expose-Headers"), "grpc-status")
	assert.Contains(t, rr.Header().Get("Access-Control-Expose-Headers"), "grpc-message")
}

func TestGrpcWebResponse_Write(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := newGrpcWebResponse(rr, false)

	resp.Header().Set("Content-Type", "application/grpc")
	resp.Header().Set("Grpc-Status", "0")

	n, err := resp.Write([]byte("test data"))

	require.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, "application/grpc-web", rr.Header().Get("Content-Type"))
	assert.Equal(t, "test data", rr.Body.String())
}

func TestGrpcWebResponse_TrailerHandling(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := newGrpcWebResponse(rr, false)

	resp.Header().Set("Content-Type", "application/grpc")
	resp.Header().Set("Grpc-Status", "0")
	resp.Header().Set("Grpc-Message", "OK")

	// Write some data
	_, err := resp.Write([]byte("response data"))
	require.NoError(t, err)

	// Simulate finishing the request
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	resp.finishRequest(req)

	body := rr.Body.Bytes()
	assert.Contains(t, string(body), "response data")

	// Check for trailer frame (should have MSB set in first byte)
	// The trailer frame should be at the end of the response
	trailerStart := len("response data")
	if len(body) > trailerStart+5 {
		trailerHeader := body[trailerStart : trailerStart+5]
		assert.Equal(t, byte(0x80), trailerHeader[0]&0x80, "Trailer frame should have MSB set")
	}
}

func TestGrpcWebResponse_TextFormat(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := newGrpcWebResponse(rr, true)

	resp.Header().Set("Content-Type", "application/grpc")
	resp.Header().Set("Grpc-Status", "0")

	_, err := resp.Write([]byte("test data"))
	require.NoError(t, err)

	resp.Flush()

	assert.Equal(t, "application/grpc-web-text", rr.Header().Get("Content-Type"))

	// Response should be base64 encoded
	decoded, err := base64.StdEncoding.DecodeString(rr.Body.String())
	require.NoError(t, err)
	assert.Equal(t, "test data", string(decoded))
}

func TestGrpcWebResponse_EmptyResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	resp := newGrpcWebResponse(rr, false)

	resp.Header().Set("Content-Type", "application/grpc")
	resp.Header().Set("Grpc-Status", "0")

	// Don't write any data, just finish the request
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	resp.finishRequest(req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/grpc-web", rr.Header().Get("Content-Type"))
}

func TestCORSOriginValidation(t *testing.T) {
	tests := []struct {
		name            string
		allowedOrigins  []string
		requestOrigin   string
		expectedAllowed bool
	}{
		{
			name:            "wildcard allows all",
			allowedOrigins:  []string{"*"},
			requestOrigin:   "https://example.com",
			expectedAllowed: true,
		},
		{
			name:            "specific origin allowed",
			allowedOrigins:  []string{"https://example.com", "https://app.example.com"},
			requestOrigin:   "https://example.com",
			expectedAllowed: true,
		},
		{
			name:            "origin not allowed",
			allowedOrigins:  []string{"https://example.com"},
			requestOrigin:   "https://evil.com",
			expectedAllowed: false,
		},
		{
			name:            "empty origins list",
			allowedOrigins:  []string{},
			requestOrigin:   "https://example.com",
			expectedAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/grpc")
				w.WriteHeader(http.StatusOK)
			})

			config := dynamic.GrpcWeb{
				AllowOrigins: tt.allowedOrigins,
			}
			middleware := New(context.Background(), next, config, "test")

			req := httptest.NewRequest(http.MethodOptions, "/test", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			req.Header.Set("Access-Control-Request-Method", "POST")
			req.Header.Set("Access-Control-Request-Headers", "x-grpc-web")

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			if tt.expectedAllowed {
				assert.Equal(t, tt.requestOrigin, rr.Header().Get("Access-Control-Allow-Origin"))
			} else {
				assert.NotEqual(t, tt.requestOrigin, rr.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestReaderCloser(t *testing.T) {
	originalBody := "test content"
	encodedBody := base64.StdEncoding.EncodeToString([]byte(originalBody))

	bodyReader := strings.NewReader(encodedBody)
	decoder := base64.NewDecoder(base64.StdEncoding, bodyReader)

	rc := &readerCloser{
		reader: decoder,
		closer: io.NopCloser(bodyReader),
	}

	// Test reading
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, originalBody, string(data))

	// Test closing
	err = rc.Close()
	assert.NoError(t, err)
}

func TestGrpcDataFrameFormat(t *testing.T) {
	// Test that trailer frames are properly formatted according to gRPC spec
	rr := httptest.NewRecorder()
	resp := newGrpcWebResponse(rr, false)

	resp.Header().Set("Content-Type", "application/grpc")
	resp.Header().Set("Grpc-Status", "0")
	resp.Header().Set("Grpc-Message", "OK")

	_, err := resp.Write([]byte("data"))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	resp.finishRequest(req)

	body := rr.Body.Bytes()

	// Find the trailer frame (should be after the data)
	dataEnd := len("data")
	if len(body) > dataEnd+5 {
		trailerFrame := body[dataEnd : dataEnd+5]

		// First byte should have MSB set (0x80) to indicate trailer
		assert.Equal(t, byte(0x80), trailerFrame[0]&0x80)

		// Next 4 bytes should be the length in big-endian format
		expectedLength := len(body) - dataEnd - 5
		actualLength := binary.BigEndian.Uint32(trailerFrame[1:5])
		assert.Equal(t, uint32(expectedLength), actualLength)
	}
}
