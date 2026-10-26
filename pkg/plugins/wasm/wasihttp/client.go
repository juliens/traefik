package wasihttp

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/traefik/traefik/v3/pkg/plugins/wasm/engine"
)

const (
	outboundDialTimeout      = 30 * time.Second
	outboundHeaderTimeout    = 30 * time.Second
	outboundMaxIdleConns     = 100
	outboundIdleConnsTimeout = 90 * time.Second
)

// newOutboundClient returns the client that sends the outbound requests of the guests of one engine.
// It follows no redirect: the guest gets the redirect response, as with a net/http.RoundTripper.
func newOutboundClient() *http.Client {
	dialer := &net.Dialer{Timeout: outboundDialTimeout, KeepAlive: 30 * time.Second}

	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          outboundMaxIdleConns,
			IdleConnTimeout:       outboundIdleConnsTimeout,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: outboundHeaderTimeout,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// startClient registers the descriptors of an outbound request and sends it on its own goroutine.
func (inst *instance) startClient(block []byte, mode uint32) uint32 {
	resp := engine.NewPipe(pipeSize, inst.notify)
	body := engine.NewPipe(pipeSize, inst.notify)

	inst.mu.Lock()
	// Ids wrap like the request ones; an id whose response descriptor is still open is skipped.
	var cid uint32
	for {
		cid = inst.nextClientID
		inst.nextClientID = (inst.nextClientID + 1) % maxClientIDs
		if _, used := inst.fds[clientFD(cid, fdClientResponseIn)]; !used {
			break
		}
	}
	inst.fds[clientFD(cid, fdClientResponseIn)] = newVFD(resp)
	if mode == bodyFromGuest {
		inst.fds[clientFD(cid, fdClientBodyOut)] = newVFD(body)
	}
	dead := inst.dead
	inst.mu.Unlock()

	if dead {
		_ = resp.CloseWithError(engine.ErrInstanceDied)

		return cid
	}

	go inst.runClient(block, mode, resp, body)

	return cid
}

// runClient sends an outbound request and writes its response, as a frame then the body, on resp.
func (inst *instance) runClient(block []byte, mode uint32, resp, body *engine.Pipe) {
	defer func() { _ = resp.Close() }()

	fail := func(msg string) {
		_, _ = resp.Write(appendFrame(appendResponse(responseBlock{Error: msg})))
	}

	if !inst.cfg.AllowOutbound {
		fail("outbound requests are not allowed: set the allowOutbound plugin setting")

		return
	}

	req, err := inst.outboundRequest(block, mode, body)
	if err != nil {
		fail(err.Error())

		return
	}

	res, err := inst.client.Do(req)
	if err != nil {
		inst.logger.Debug().Err(err).Str("url", req.URL.Redacted()).Msg("Wasm outbound request failed")
		fail(err.Error())

		return
	}
	defer func() { _ = res.Body.Close() }()

	frame := appendFrame(appendResponse(responseBlock{Status: res.StatusCode, Header: res.Header}))
	if _, err = resp.Write(frame); err != nil {
		return
	}

	// A write error means that the guest closed the response, or that the instance stopped: closing the body aborts
	// the request.
	_, _ = io.Copy(resp, res.Body)
}

// outboundRequest builds the request described by the guest; it is canceled when the instance stops.
func (inst *instance) outboundRequest(block []byte, mode uint32, body *engine.Pipe) (*http.Request, error) {
	info, err := parseRequest(block)
	if err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if info.Scheme != "http" && info.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q", info.Scheme)
	}

	var reqBody io.Reader = http.NoBody
	if mode == bodyFromGuest {
		reqBody = pipeReader{ctx: inst.ctx, pipe: body}
	}

	req, err := http.NewRequestWithContext(inst.ctx, info.Method, info.Scheme+"://"+info.Host+info.Target, reqBody)
	if err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	req.Header = info.Header
	if host := info.Header.Get("Host"); host != "" {
		req.Host = host
		req.Header.Del("Host")
	}
	if mode == bodyFromGuest {
		req.ContentLength = info.ContentLength
	}

	return req, nil
}
