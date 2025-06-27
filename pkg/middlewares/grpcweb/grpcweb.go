package grpcweb

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/cors"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/middlewares"
)

const (
	// typeName is the name used for logging and identification
	typeName = "GRPCWeb"

	// Content types as defined in the gRPC-Web protocol specification
	// https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md#protocol-differences-vs-grpc-over-http2
	grpcContentType        = "application/grpc"
	grpcWebContentType     = "application/grpc-web"
	grpcWebTextContentType = "application/grpc-web-text"
)

type Middleware struct {
	next        http.Handler
	corsWrapper *cors.Cors
}

// New builds a new gRPC web request converter.
func New(ctx context.Context, next http.Handler, config dynamic.GrpcWeb, name string) Middleware {
	middlewares.GetLogger(ctx, name, typeName).Debug().Msg("Creating Middleware")

	corsWrapper := cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool {
			for _, originCfg := range config.AllowOrigins {
				if originCfg == "*" || originCfg == origin {
					return true
				}
			}
			return false
		},
		AllowedHeaders:   []string{"*", "U-A"},
		ExposedHeaders:   nil,  // make sure that this is *nil*, otherwise the WebResponse overwrite will not work.
		AllowCredentials: true, // always allow credentials, otherwise :authorization headers won't work
		MaxAge:           int(10 * time.Minute),
	})

	return Middleware{next: next, corsWrapper: corsWrapper}
}

func (m Middleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	accessControlHeaders := strings.ToLower(req.Header.Get("Access-Control-Request-Headers"))
	if (req.Method == http.MethodOptions && strings.Contains(accessControlHeaders, "x-grpc-web")) ||
		(req.Method == http.MethodPost && strings.HasPrefix(req.Header.Get("Content-Type"), grpcWebContentType)) {

		m.corsWrapper.Handler(http.HandlerFunc(m.HandleGrpcWebRequest)).ServeHTTP(rw, req)
		return
	}
	m.next.ServeHTTP(rw, req)
}

// HandleGrpcWebRequest takes a HTTP request that is assumed to be a gRPC-Web request and wraps it with a compatibility
// layer to transform it to a standard gRPC request for the wrapped gRPC server and transforms the response to comply
// with the gRPC-Web protocol.
func (m Middleware) HandleGrpcWebRequest(resp http.ResponseWriter, req *http.Request) {
	intReq, isTextFormat := convertToGrpcRequest(req)
	intResp := newGrpcWebResponse(resp, isTextFormat)
	intReq.URL.Path = req.URL.Path
	m.next.ServeHTTP(intResp, intReq)
	intResp.finishRequest(req)
}

func convertToGrpcRequest(req *http.Request) (*http.Request, bool) {
	// Create a shallow copy of the request to avoid mutating the original
	newReq := *req
	newReq.Header = req.Header.Clone()
	newReq.ProtoMajor = 2
	newReq.ProtoMinor = 0

	contentType := newReq.Header.Get("Content-Type")
	incomingContentType := grpcWebContentType
	isTextFormat := strings.HasPrefix(contentType, grpcWebTextContentType)
	if isTextFormat {
		// body is base64-encoded: decode it; Wrap it in readerCloser so Body is still closed
		decoder := base64.NewDecoder(base64.StdEncoding, newReq.Body)
		newReq.Body = &readerCloser{reader: decoder, closer: newReq.Body}
		incomingContentType = grpcWebTextContentType
	}
	newReq.Header.Set("Content-Type", strings.Replace(contentType, incomingContentType, grpcContentType, 1))

	// Remove content-length header since it represents http1.1 payload size, not the sum of the h2
	// DATA frame payload lengths. https://http2.github.io/http2-spec/#malformed This effectively
	// switches to chunked encoding which is the default for h2
	newReq.Header.Del("Content-Length")
	newReq.ContentLength = -1

	return &newReq, isTextFormat
}

// readerCloser combines an io.Reader and an io.Closer into an io.ReadCloser.
// This is used to wrap a base64 decoder while preserving the ability to close the original body.
type readerCloser struct {
	reader io.Reader
	closer io.Closer
}

func (rc *readerCloser) Read(dest []byte) (int, error) {
	return rc.reader.Read(dest)
}

func (rc *readerCloser) Close() error {
	return rc.closer.Close()
}
