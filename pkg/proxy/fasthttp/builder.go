package fasthttp

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
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
func (r *ProxyBuilder) Build(cfgName string, cfg *dynamic.HTTPClientConfig, tlsConfig *tls.Config, target *url.URL) (http.Handler, error) {
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

	r.pools[cfgName][target.String()] = connPool
	return connPool
}

func getDialFn(realTarget *url.URL, tlsConfig *tls.Config, config *dynamic.HTTPClientConfig) func() (net.Conn, error) {
	proxyURL, err := http.ProxyFromEnvironment(&http.Request{URL: realTarget})
	if err != nil {
		return func() (net.Conn, error) {
			return nil, err
		}
	}

	realTargetAddr := addrFromURL(realTarget)

	proxyDialer := getDialer(proxyURL.Scheme, tlsConfig, config)
	proxyAddr := addrFromURL(proxyURL)

	switch {
	case proxyURL == nil:
		// Nothing to do.

	case proxyURL.Scheme == "socks5":
		var auth *proxy.Auth
		if u := proxyURL.User; u != nil {
			auth = &proxy.Auth{User: u.Username()}
			auth.Password, _ = u.Password()
		}

		// SOCKS5 implementation do not return errors.
		socksDialer, _ := proxy.SOCKS5("tcp", proxyAddr, auth, proxyDialer)

		return func() (net.Conn, error) {
			co, err := socksDialer.Dial("tcp", realTargetAddr)
			if err != nil {
				return nil, err
			}

			if realTarget.Scheme == "https" {
				c := &tls.Config{}
				if tlsConfig != nil {
					c = tlsConfig.Clone()
				}

				if c.ServerName == "" {
					c.ServerName = realTarget.Hostname()
				}
				return tls.Client(co, c), nil
			}
			return co, nil
		}

	case realTarget.Scheme == "http":
		// Nothing to do the Proxy-Authorization header will be added by the ReverseProxy.

	case realTarget.Scheme == "https":
		hdr := make(http.Header)
		if u := proxyURL.User; u != nil {
			username := u.Username()
			password, _ := u.Password()
			auth := username + ":" + password
			hdr.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
		}

		return func() (net.Conn, error) {
			conn, err := proxyDialer.Dial("tcp", proxyAddr)
			if err != nil {
				return nil, err
			}

			connectReq := &http.Request{
				Method: http.MethodConnect,
				URL:    &url.URL{Opaque: realTargetAddr},
				Host:   realTarget.Host,
				Header: make(http.Header),
			}

			// If there's no done channel (no deadline or cancellation
			// from the caller possible), at least set some (long)
			// timeout here. This will make sure we don't block forever
			// and leak a goroutine if the connection stops replying
			// after the TCP connect.
			// FIXME comment
			connectCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			didReadResponse := make(chan struct{}) // closed after CONNECT write+read is done or fails
			var resp *http.Response

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
				resp, err = http.ReadResponse(br, connectReq)
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

			if realTarget.Scheme == "https" {
				c := &tls.Config{}
				if tlsConfig != nil {
					c = tlsConfig.Clone()
				}

				if c.ServerName == "" {
					c.ServerName = realTarget.Hostname()
				}
				return tls.Client(conn, c), nil
			}
			return conn, nil
		}
	}

	return func() (net.Conn, error) {
		d := getDialer(realTarget.Scheme, tlsConfig, config)
		return d.Dial("tcp", realTargetAddr)
	}

}

func addrFromURL(u *url.URL) string {
	addr := u.Host

	if u.Port() != "" {
		if u.Scheme == "http" {
			return addr + ":80"
		}
		if u.Scheme == "https" {
			return addr + ":443"
		}
	}

	return addr
}

type Dialer interface {
	Dial(network, addr string) (c net.Conn, err error)
}

type DialerFunc func(network, addr string) (c net.Conn, err error)

func (d DialerFunc) Dial(network, addr string) (c net.Conn, err error) {
	return d(network, addr)
}

func getDialer(scheme string, tlsConfig *tls.Config, cfg *dynamic.HTTPClientConfig) Dialer {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if cfg.ForwardingTimeouts != nil {
		dialer.Timeout = time.Duration(cfg.ForwardingTimeouts.DialTimeout)
	}

	if scheme == "https" && tlsConfig != nil {
		return DialerFunc(func(network, addr string) (c net.Conn, err error) {
			return tls.DialWithDialer(dialer, network, addr, tlsConfig)
		})
	}
	return dialer
}
