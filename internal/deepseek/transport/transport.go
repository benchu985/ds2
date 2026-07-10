package transport

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type DialContextFunc func(ctx context.Context, network, addr string) (net.Conn, error)

type Client struct {
	http *http.Client
}

func New(timeout time.Duration) *Client {
	return NewWithDialContext(timeout, nil)
}

func NewWithDialContext(timeout time.Duration, dialContext DialContextFunc) *Client {
	useEnvProxy := dialContext == nil
	if dialContext == nil {
		dialContext = (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	}
	base := &http.Transport{
		ForceAttemptHTTP2:   false,
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DialContext:         dialContext,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
	}
	if useEnvProxy {
		base.Proxy = http.ProxyFromEnvironment
	}
	return &Client{http: &http.Client{Timeout: timeout, Transport: base}}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.http.Do(req)
}

func NewFallbackClient(timeout time.Duration, dialContext DialContextFunc) *http.Client {
	useEnvProxy := dialContext == nil
	if dialContext == nil {
		dialContext = (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	}
	base := &http.Transport{
		ForceAttemptHTTP2:   false,
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DialContext:         dialContext,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
	}
	if useEnvProxy {
		base.Proxy = http.ProxyFromEnvironment
	}
	return &http.Client{Timeout: timeout, Transport: base}
}
