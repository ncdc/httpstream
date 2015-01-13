package spdy

import (
	"net/http"

	"github.com/docker/spdystream"
	"github.com/golang/glog"
	"github.com/ncdc/httpstream"
)

type spdy31Connection struct {
	conn *spdystream.Connection
}

func (c *spdy31Connection) Close() error {
	glog.Info("conn close")
	return c.conn.Close()
}

func (c *spdy31Connection) CloseWait() error {
	return c.conn.CloseWait()
}

func (c *spdy31Connection) CreateStream(headers http.Header) (httpstream.Stream, error) {
	stream, err := c.conn.CreateStream(headers, nil, false)
	if err != nil {
		return nil, err
	}
	return &spdy31Stream{stream: stream}, nil
}
