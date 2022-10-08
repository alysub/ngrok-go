package ngrok

import (
	"context"
	"net"
	"net/http"
	"time"

	tunnel_client "github.com/ngrok/ngrok-go/internal/tunnel/client"
)

// An ngrok tunnel.
type Tunnel interface {
	// Every Tunnel is a net.Listener. It can be plugged into any existing
	// code that expects a net.Listener seamlessly without any changes.
	net.Listener

	// Returns the ForwardsTo string for this tunnel.
	ForwardsTo() string
	// Returns the Metadata string for this tunnel.
	Metadata() string
	// Returns this tunnel's ID.
	ID() string

	// Returns this tunnel's protocol.
	// Will be empty for labeled tunnels.
	Proto() string
	// Returns the URL for this tunnel.
	// Will be empty for labeled tunnels.
	URL() string

	// Returns the labels for this tunnel.
	// Will be empty for non-labeled tunnels.
	Labels() map[string]string

	// Session returns the tunnel's parent Session object that it
	// was started on.
	Session() Session

	// Use this tunnel to serve HTTP requests.
	AsHTTP() HTTPTunnel

	// Convenience method that calls `CloseWithContext` with a default timeout
	// of 5 seconds.
	Close() error
	// Closing a tunnel is an operation that involves sending a "close" message
	// over the existing session. Since this is subject to network latency,
	// packet loss, etc., it is most correct to provide a context. See also
	// `Close`, which matches the `io.Closer` interface method.
	CloseWithContext(context.Context) error
}

// A tunnel that may be used to serve HTTP.
type HTTPTunnel interface {
	Tunnel
	// Serve HTTP requests over this tunnel using the provided [http.Handler].
	Serve(context.Context, http.Handler) error
}

type tunnelImpl struct {
	Sess   Session
	Tunnel tunnel_client.Tunnel
}

func (t *tunnelImpl) Accept() (net.Conn, error) {
	conn, err := t.Tunnel.Accept()
	if err != nil {
		return nil, errAcceptFailed{Inner: err}
	}
	return &connImpl{
		Conn:  conn.Conn,
		Proxy: conn,
	}, nil
}

func (t *tunnelImpl) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	return t.CloseWithContext(ctx)
}

func (t *tunnelImpl) CloseWithContext(_ context.Context) error {
	return t.Tunnel.Close()
}

func (t *tunnelImpl) Addr() net.Addr {
	return t.Tunnel.Addr()
}

func (t *tunnelImpl) URL() string {
	return t.Tunnel.RemoteBindConfig().URL
}

func (t *tunnelImpl) Proto() string {
	return t.Tunnel.RemoteBindConfig().ConfigProto
}

func (t *tunnelImpl) ForwardsTo() string {
	return t.Tunnel.ForwardsTo()
}

func (t *tunnelImpl) Metadata() string {
	return t.Tunnel.RemoteBindConfig().Metadata
}

func (t *tunnelImpl) ID() string {
	return t.Tunnel.ID()
}

func (t *tunnelImpl) Labels() map[string]string {
	return t.Tunnel.RemoteBindConfig().Labels
}

func (t *tunnelImpl) AsHTTP() HTTPTunnel {
	return t
}

func (t *tunnelImpl) Session() Session {
	return t.Sess
}

func (t *tunnelImpl) Serve(ctx context.Context, h http.Handler) error {
	srv := http.Server{
		Handler:     h,
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}
	return srv.Serve(t)
}

type connImpl struct {
	net.Conn
	Proxy *tunnel_client.ProxyConn
}

func (c *connImpl) ProxyConn() *tunnel_client.ProxyConn {
	return c.Proxy
}
