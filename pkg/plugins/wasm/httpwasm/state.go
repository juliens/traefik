package httpwasm

import (
	"context"
	"fmt"
	"io"
	"net/http"

	handlerapi "github.com/http-wasm/http-wasm-host-go/api/handler"
)

// requestStateKey is a context.Context value associated with a requestState pointer to the current request.
type requestStateKey struct{}

func requestStateFromContext(ctx context.Context) *requestState {
	return ctx.Value(requestStateKey{}).(*requestState)
}

// requestState is the state of one request, read by the host functions through the call context.
type requestState struct {
	w    http.ResponseWriter
	r    *http.Request
	next http.Handler

	afterNext          bool
	requestBodyReader  io.ReadCloser
	requestBodyWriter  io.Writer
	responseBodyReader io.ReadCloser
	responseBodyWriter io.Writer

	// features are the current request's features which may be more than the instance ones.
	features handlerapi.Features
}

func newRequestState(w http.ResponseWriter, r *http.Request, next http.Handler, features handlerapi.Features) *requestState {
	s := &requestState{w: w, r: r, next: next}
	s.enableFeatures(features)

	return s
}

func (s *requestState) enableFeatures(features handlerapi.Features) {
	s.features = s.features.WithEnabled(features)
	if features.IsEnabled(handlerapi.FeatureBufferRequest) {
		if _, ok := s.r.Body.(*bufferingRequestBody); !ok {
			s.r.Body = &bufferingRequestBody{delegate: s.r.Body}
		}
	}
	if s.features.IsEnabled(handlerapi.FeatureBufferResponse) {
		if _, ok := s.w.(*bufferingResponseWriter); !ok {
			s.w = &bufferingResponseWriter{delegate: s.w}
		}
	}
}

// handleNext calls the next handler, recovering its panics as errors.
func (s *requestState) handleNext() (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if e, ok := recovered.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", recovered)
			}
		}
	}()

	// If the request body was intercepted for any reason, reset it before calling downstream.
	if br, ok := s.r.Body.(*bufferingRequestBody); ok {
		if br.buffer.Len() == 0 {
			s.r.Body = br.delegate
		} else {
			_ = br.Close()
			s.r.Body = io.NopCloser(&br.buffer)
		}
	}

	s.next.ServeHTTP(s.w, s.r)

	return nil
}

// closeRequest releases the request body resources, before the next handler runs.
func (s *requestState) closeRequest() error {
	if reqBW := s.requestBodyWriter; reqBW != nil {
		if f, ok := reqBW.(http.Flusher); ok {
			f.Flush()
		}
		s.requestBodyWriter = nil
	}

	var err error
	if reqBR := s.requestBodyReader; reqBR != nil {
		err = reqBR.Close()
		s.requestBodyReader = nil
	}

	return err
}

// close releases all the resources of the request.
func (s *requestState) close() error {
	err := s.closeRequest()

	if respBW := s.responseBodyWriter; respBW != nil {
		if f, ok := respBW.(http.Flusher); ok {
			f.Flush()
		}
		s.responseBodyWriter = nil
	}

	if respBR := s.responseBodyReader; respBR != nil {
		if closeErr := respBR.Close(); err == nil {
			err = closeErr
		}
		s.responseBodyReader = nil
	}

	return err
}
