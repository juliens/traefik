package fasthttp

import (
	"bufio"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"golang.org/x/net/internal/socks"
	"golang.org/x/net/proxy"
)

// ProxyBuilder handles the connection pools for the FastHTTP proxies.
type ProxyBuilder struct {
	// lock isn't needed because ProxyBuilder is not called concurrently.
	pools map[string]map[string]*ConnPool
}

// NewProxyBuilder creates a new ProxyBuilder.
func NewProxyBuilder() *ProxyBuilder {
	return &ProxyBuilder{
		pools: make(map[string]map[string]*ConnPool),
	}
}

// Delete deletes the round-tripper corresponding to the given dynamic.HTTPClientConfig.
func (r *ProxyBuilder) Delete(cfgName string) {
	delete(r.pools, cfgName)
}

// Build builds a new ReverseProxy with the given configuration.
func (r *ProxyBuilder) Build(cfgName string, cfg *dynamic.HTTPClientConfig, tlsConfig *tls.Config, target *url.URL) http.Handler {
	pool := r.getPool(cfgName, cfg, tlsConfig, target)

	return NewReverseProxy(target, cfg.PassHostHeader, pool)
}

func (r *ProxyBuilder) getPool(cfgName string, config *dynamic.HTTPClientConfig, tlsConfig *tls.Config, target *url.URL) *ConnPool {
	addr := target.Host
	if target.Port() == "" {
		if target.Scheme == "https" {
			addr += ":443"
		} else {
			addr += ":80"
		}
	}


	pool, ok := r.pools[cfgName]
	if !ok {
		pool = make(map[string]*ConnPool)
		r.pools[cfgName] = pool
	}

	if connPool, ok := pool[target.String()]; ok {
		return connPool
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}


	if config.ForwardingTimeouts != nil {
		dialer.Timeout = time.Duration(config.ForwardingTimeouts.DialTimeout)
	}

	idleConnTimeout := time.Duration(dynamic.DefaultIdleConnTimeout)
	if config.ForwardingTimeouts != nil {
		idleConnTimeout = time.Duration(config.ForwardingTimeouts.IdleConnTimeout)
	}

	dialFn := getDialFn(target, tlsConfig, config)

	connPool := NewConnPool(config.MaxIdleConnsPerHost, idleConnTimeout, dialFn)

	connPool.modifyRequest = getModifyRequest(target, tlsConfig)
	r.pools[cfgName][target.String()] = connPool
	return connPool
}

func getModifyRequest(target *url.URL, config *tls.Config) func(*http.Request) {

}

func getDialFn(realTarget *url.URL, tlsConfig *tls.Config, config *dynamic.HTTPClientConfig) func() (net.Conn, error) {
	dialProxyURL, err := http.ProxyFromEnvironment(&http.Request{
		URL: realTarget,
	})
	if err != nil {
		return func() (net.Conn, error) {
			return nil, err
		}
	}

	target := realTarget

	if dialProxyURL != nil {
		target = dialProxyURL
	}


	addr := target.Host
	if target.Port() == "" {
		if target.Scheme == "https" {
			addr += ":443"
		} else {
			addr += ":80"
		}
	}



	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if config.ForwardingTimeouts != nil {
		dialer.Timeout = time.Duration(config.ForwardingTimeouts.DialTimeout)
	}

	if dialProxyURL.Scheme == "socks5" {
		var auth *proxy.Auth
		if u := dialProxyURL.User; u != nil {
			auth = &proxy.Auth{
				User: u.Username(),
			}
			auth.Password, _ = u.Password()
		}

		// never returns errors
		dialerSock, _ := proxy.SOCKS5("tcp", addr, auth, dialer)

		return func() (net.Conn, error) {
			conn, err :=  dialerSock.Dial("tcp", addr)

			if tlsConfig != nil {
				if tlsConfig.ServerName == "" {
					// Make a copy to avoid polluting argument or default.
					c := tlsConfig.Clone()
					// FIXME
					c.ServerName = realTarget.Hostname()
					tlsConfig = c
				}
				return tls.Client(conn, tlsConfig), err
			}
			return conn, err
		}
	}


	if tlsConfig != nil {
		return func() (net.Conn, error) {
			return tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		}
	}
	return func() (net.Conn, error) {
		return dialer.Dial("tcp", addr)
	}
}

func getDialer(target *url.URL, tlsConfig *tls.Config, config *dynamic.HTTPClientConfig) func() (net.Conn, error) {
	dialProxyURL, err := http.ProxyFromEnvironment(&http.Request{
		URL: target,
	})
	if err != nil {
		return func() (net.Conn, error) {
			return nil, err
		}
	}
	if dialProxyURL != nil {
		target = dialProxyURL
	}

	addr := target.Host
	if target.Port() == "" {
		if target.Scheme == "https" {
			addr += ":443"
		} else {
			addr += ":80"
		}
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if config.ForwardingTimeouts != nil {
		dialer.Timeout = time.Duration(config.ForwardingTimeouts.DialTimeout)
	}

	dialFn := dialer.Dial

	if tlsConfig != nil {
		return tls.Dial("tcp", addr, tlsConfig)
	}
	return dialFn
}
}

func getProxyDialer(proxyUrl) func() (net.Conn, error) {
	switch {
	case cm.proxyURL == nil:
		// Do nothing. Not using a proxy.
	case cm.proxyURL.Scheme == "socks5":
		conn := pconn.conn
		d := socksNewDialer("tcp", conn.RemoteAddr().String())
		if u := cm.proxyURL.User; u != nil {
			auth := &socksUsernamePassword{
				Username: u.Username(),
			}
			auth.Password, _ = u.Password()
			d.AuthMethods = []socksAuthMethod{
				socksAuthMethodNotRequired,
				socksAuthMethodUsernamePassword,
			}
			d.Authenticate = auth.Authenticate
		}
		if _, err := d.DialWithConn(ctx, conn, "tcp", cm.targetAddr); err != nil {
			conn.Close()
			return nil, err
		}
	case cm.targetScheme == "http":
		pconn.isProxy = true
		if pa := cm.proxyAuth(); pa != "" {
			pconn.mutateHeaderFunc = func(h Header) {
				h.Set("Proxy-Authorization", pa)
			}
		}
	case cm.targetScheme == "https":
		conn := pconn.conn
		var hdr Header
		if t.GetProxyConnectHeader != nil {
			var err error
			hdr, err = t.GetProxyConnectHeader(ctx, cm.proxyURL, cm.targetAddr)
			if err != nil {
				conn.Close()
				return nil, err
			}
		} else {
			hdr = t.ProxyConnectHeader
		}
		if hdr == nil {
			hdr = make(Header)
		}
		if pa := cm.proxyAuth(); pa != "" {
			hdr = hdr.Clone()
			hdr.Set("Proxy-Authorization", pa)
		}
		connectReq := &Request{
			Method: "CONNECT",
			URL:    &url.URL{Opaque: cm.targetAddr},
			Host:   cm.targetAddr,
			Header: hdr,
		}

		// If there's no done channel (no deadline or cancellation
		// from the caller possible), at least set some (long)
		// timeout here. This will make sure we don't block forever
		// and leak a goroutine if the connection stops replying
		// after the TCP connect.
		connectCtx := ctx
		if ctx.Done() == nil {
			newCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
			defer cancel()
			connectCtx = newCtx
		}

		didReadResponse := make(chan struct{}) // closed after CONNECT write+read is done or fails
		var (
			resp *Response
			err  error // write or read error
		)
		// Write the CONNECT request & read the response.
		go func() {
			defer close(didReadResponse)
			err = connectReq.Write(conn)
			if err != nil {
				return
			}
			// Okay to use and discard buffered reader here, because
			// TLS server will not speak until spoken to.
			br := bufio.NewReader(conn)
			resp, err = ReadResponse(br, connectReq)
		}()
		select {
		case <-connectCtx.Done():
			conn.Close()
			<-didReadResponse
			return nil, connectCtx.Err()
		case <-didReadResponse:
			// resp or err now set
		}
		if err != nil {
			conn.Close()
			return nil, err
		}
		if resp.StatusCode != 200 {
			_, text, ok := strings.Cut(resp.Status, " ")
			conn.Close()
			if !ok {
				return nil, errors.New("unknown status code")
			}
			return nil, errors.New(text)
		}
	}

}
