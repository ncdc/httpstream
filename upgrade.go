package andyspdy

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/docker/spdystream"
)

type RequestUpgrader interface {
	Upgrade(request *http.Request) (*UpgradedRequest, error)
}

type UpgradedRequest interface {
	io.Closer
	CreateStream() (*Stream, error)
}

type spdy31UpgradedRequest struct {
	conn *spdystream.Connection
}

func (c *spdy31UpgradedRequest) Close() error {
	return c.conn.Close()
}

func (c *spdy31UpgradedRequest) CreateStream(headers http.Header) (*Stream, error) {
	stream, err := c.conn.CreateStream(headers, nil, false)
	if err != nil {
		return nil, err
	}
	return &spdy31Stream{stream: stream}
}

type Stream interface {
	io.ReadWriteCloser
}

type spdy31Stream struct {
	stream *spdystream.Stream
}

func (s *spdy31Stream) Read(p []byte) (n int, err error) {
	return s.stream.Read(p)
}

func (s *spdy31Stream) Write(data []byte) (n int, err error) {
	return s.stream.Write(p)
}

func (s *spdy31Stream) Close() error {
	return s.stream.Close()
}

func Upgrade(req *http.Request) (*UpgradedRequest, error) {
	req.Header.Add("Connection", "Upgrade")
	req.Header.Add("Upgrade", "SPDY/3.1")

	conn, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	err = req.Write(conn)
	if err != nil {
		return nil, err
	}

	resp, err := http.ReadResponse(conn, req)
	if err != nil {
		return nil, err
	}

	if resp.Header.Get("Connection") != "Upgrade" || resp.Header.Get("Upgrade") != "SPDY/3.1" {
		return nil, fmt.Errorf("Expected upgrade to SPDY/3.1 from server; got %#v instead", resp.Header)
	}

	spdyConn, err := spdystream.NewConnection(conn, false)
	if err != nil {
		return nil, err
	}

	return &spdy31UpgradedRequest{conn: spdyConn}, nil
}
