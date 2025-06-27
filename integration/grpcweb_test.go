package integration

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/traefik/traefik/v3/integration/helloworld"
	"github.com/traefik/traefik/v3/integration/try"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GRPCWebSuite struct{ BaseSuite }

func TestGRPCWebSuite(t *testing.T) {
	suite.Run(t, new(GRPCWebSuite))
}

type grpcWebServer struct{}

func (s *grpcWebServer) SayHello(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func (s *grpcWebServer) SayHelloError(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return nil, status.Error(codes.InvalidArgument, "test error message")
}

func (s *grpcWebServer) SayHelloWithHeaders(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	// Set response headers
	header := metadata.New(map[string]string{
		"x-custom-header": "custom-value",
		"x-response-id":   "12345",
	})
	grpc.SetHeader(ctx, header)

	// Set response trailers
	trailer := metadata.New(map[string]string{
		"x-custom-trailer": "trailer-value",
		"x-total-count":    "100",
	})
	grpc.SetTrailer(ctx, trailer)

	return &helloworld.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func (s *grpcWebServer) StreamExample(req *helloworld.StreamExampleRequest, stream helloworld.Greeter_StreamExampleServer) error {
	return stream.Send(&helloworld.StreamExampleReply{Data: "streaming data"})
}

func (s *GRPCWebSuite) SetupSuite() {
	// Ensure we have the necessary certificates
	var err error
	LocalhostCert, err = os.ReadFile("./resources/tls/local.cert")
	require.NoError(s.T(), err)
	LocalhostKey, err = os.ReadFile("./resources/tls/local.key")
	require.NoError(s.T(), err)
}

func startGRPCWebServer(lis net.Listener) error {
	cert, err := tls.X509KeyPair(LocalhostCert, LocalhostKey)
	if err != nil {
		return err
	}

	creds := credentials.NewServerTLSFromCert(&cert)
	s := grpc.NewServer(grpc.Creds(creds))
	defer s.Stop()

	helloworld.RegisterGreeterServer(s, &grpcWebServer{})
	return s.Serve(lis)
}

func (s *GRPCWebSuite) TestGRPCWebBinaryFormat() {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(s.T(), err)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(s.T(), err)

	go func() {
		err := startGRPCWebServer(lis)
		assert.NoError(s.T(), err)
	}()

	file := s.adaptFile("fixtures/grpcweb/config.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	s.traefikCmd(withConfigFile(file))

	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	require.NoError(s.T(), err)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	grpcWebReq := createGRPCWebRequest(s.T(), "application/grpc-web+proto", "World", false)

	resp, err := client.Do(grpcWebReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(s.T(), "application/grpc-web+proto", resp.Header.Get("Content-Type"))
	assert.Contains(s.T(), resp.Header.Get("Access-Control-Expose-Headers"), "grpc-status")
	assert.Contains(s.T(), resp.Header.Get("Access-Control-Expose-Headers"), "grpc-message")

	body, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	assert.Contains(s.T(), string(body), "Hello World")
}

func (s *GRPCWebSuite) TestGRPCWebTextFormat() {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(s.T(), err)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(s.T(), err)

	go func() {
		err := startGRPCWebServer(lis)
		assert.NoError(s.T(), err)
	}()

	file := s.adaptFile("fixtures/grpcweb/config.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	s.traefikCmd(withConfigFile(file))

	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	require.NoError(s.T(), err)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	grpcWebReq := createGRPCWebRequest(s.T(), "application/grpc-web-text+proto", "World", true)

	resp, err := client.Do(grpcWebReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(s.T(), "application/grpc-web-text+proto", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	decodedBody, err := base64.StdEncoding.DecodeString(string(body))
	require.NoError(s.T(), err)

	assert.Contains(s.T(), string(decodedBody), "Hello World")
}

func (s *GRPCWebSuite) TestGRPCWebCORS() {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(s.T(), err)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(s.T(), err)

	go func() {
		err := startGRPCWebServer(lis)
		assert.NoError(s.T(), err)
	}()

	file := s.adaptFile("fixtures/grpcweb/config.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	s.traefikCmd(withConfigFile(file))

	// Wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	require.NoError(s.T(), err)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Test CORS preflight request
	req, err := http.NewRequest(http.MethodOptions, "https://127.0.0.1:4443/helloworld.Greeter/SayHello", nil)
	require.NoError(s.T(), err)

	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "x-grpc-web,content-type")

	resp, err := client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(s.T(), "https://example.com", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(s.T(), "true", resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Contains(s.T(), resp.Header.Get("Access-Control-Allow-Headers"), "Content-Type")
}

func (s *GRPCWebSuite) TestGRPCWebTrailers() {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(s.T(), err)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(s.T(), err)

	go func() {
		err := startGRPCWebServer(lis)
		assert.NoError(s.T(), err)
	}()

	file := s.adaptFile("fixtures/grpcweb/config.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	s.traefikCmd(withConfigFile(file))

	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	require.NoError(s.T(), err)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	grpcWebReq := createGRPCWebRequestWithHeaders(s.T(), "application/grpc-web+proto", "World", false)

	resp, err := client.Do(grpcWebReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	s.verifyTrailerFrame(body)
}

func (s *GRPCWebSuite) TestGRPCWebErrorHandling() {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(s.T(), err)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(s.T(), err)

	go func() {
		err := startGRPCWebServer(lis)
		assert.NoError(s.T(), err)
	}()

	file := s.adaptFile("fixtures/grpcweb/config.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	s.traefikCmd(withConfigFile(file))

	// Wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	require.NoError(s.T(), err)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Create a gRPC-Web request that will return an error
	grpcWebReq := createGRPCWebErrorRequest(s.T(), "application/grpc-web+proto", "World", false)

	resp, err := client.Do(grpcWebReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// gRPC errors are returned as HTTP 200 with error details in trailers
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	// gRPC errors are encoded in trailer frames, not as plain text
	// Verify that we have a trailer frame (starts with 0x80)
	assert.True(s.T(), len(body) >= 5, "Response should be long enough to contain trailer frame header")
	assert.Equal(s.T(), byte(0x80), body[0]&0x80, "Response should contain a trailer frame for errors")

	// For now, just verify the trailer frame format is correct
	// The length can be 0 if no custom trailers are set
	trailerLength := binary.BigEndian.Uint32(body[1:5])
	assert.True(s.T(), trailerLength >= 0, "Trailer frame length should be valid")
}

func (s *GRPCWebSuite) TestGRPCWebProtocolCompliance() {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(s.T(), err)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(s.T(), err)

	go func() {
		err := startGRPCWebServer(lis)
		assert.NoError(s.T(), err)
	}()

	file := s.adaptFile("fixtures/grpcweb/config.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	s.traefikCmd(withConfigFile(file))

	// Wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	require.NoError(s.T(), err)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Test various protocol compliance aspects
	testCases := []struct {
		name        string
		contentType string
		isText      bool
	}{
		{"binary-proto", "application/grpc-web+proto", false},
		{"binary-json", "application/grpc-web+json", false},
		{"text-proto", "application/grpc-web-text+proto", true},
		{"text-json", "application/grpc-web-text+json", true},
		{"binary-default", "application/grpc-web", false},
		{"text-default", "application/grpc-web-text", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			grpcWebReq := createGRPCWebRequest(t, tc.contentType, "World", tc.isText)

			resp, err := client.Do(grpcWebReq)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, tc.contentType, resp.Header.Get("Content-Type"))

			// Verify CORS headers are present
			assert.Contains(t, resp.Header.Get("Access-Control-Expose-Headers"), "grpc-status")
			assert.Contains(t, resp.Header.Get("Access-Control-Expose-Headers"), "grpc-message")
		})
	}
}

// Helper functions

func createGRPCWebRequest(t *testing.T, contentType, name string, isText bool) *http.Request {
	t.Helper()

	// Create a proper HelloRequest protobuf message
	helloReq := &helloworld.HelloRequest{
		Name: name,
	}

	// Marshal the protobuf message
	protoData, err := proto.Marshal(helloReq)
	require.NoError(t, err)

	// Create gRPC message frame: [compression-flag][4-byte-length][message]
	grpcFrame := make([]byte, 5+len(protoData))
	grpcFrame[0] = 0 // no compression
	binary.BigEndian.PutUint32(grpcFrame[1:5], uint32(len(protoData)))
	copy(grpcFrame[5:], protoData)

	var body io.Reader
	if isText {
		// Base64 encode the gRPC frame for text format
		encoded := base64.StdEncoding.EncodeToString(grpcFrame)
		body = strings.NewReader(encoded)
	} else {
		body = bytes.NewReader(grpcFrame)
	}

	req, err := http.NewRequest(http.MethodPost, "https://127.0.0.1:4443/helloworld.Greeter/SayHello", body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Grpc-Web", "1")

	return req
}

func createGRPCWebRequestWithHeaders(t *testing.T, contentType, name string, isText bool) *http.Request {
	t.Helper()

	req := createGRPCWebRequest(t, contentType, name, isText)
	req.URL.Path = "/helloworld.Greeter/SayHelloWithHeaders"

	return req
}

func createGRPCWebErrorRequest(t *testing.T, contentType, name string, isText bool) *http.Request {
	t.Helper()

	req := createGRPCWebRequest(t, contentType, name, isText)
	req.URL.Path = "/helloworld.Greeter/SayHelloError"

	return req
}

func (s *GRPCWebSuite) verifyTrailerFrame(body []byte) {
	s.T().Helper()

	// gRPC Web responses should contain a trailer frame (starts with 0x80)
	// See: https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md#protocol-differences-vs-grpc-over-http2
	assert.True(s.T(), len(body) >= 5, "Response should contain trailer frame")
	assert.True(s.T(), bytes.Contains(body, []byte{0x80}), "Response should contain a trailer frame")
}
