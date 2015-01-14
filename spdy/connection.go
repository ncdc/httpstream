package spdy

import (
	"net/http"
	"sync"
	"time"

	"github.com/docker/spdystream"
	"github.com/ncdc/httpstream"
)

type spdy31Connection struct {
	conn *spdystream.Connection
	wg   sync.WaitGroup
}

func (c *spdy31Connection) Close() error {
	return c.conn.Close()
}

func (c *spdy31Connection) CloseWait() error {
	// wait until all the streams have been closed
	c.wg.Wait()

	// now close the connection
	return c.conn.CloseWait()
}

func (c *spdy31Connection) CreateStream(headers http.Header) (httpstream.Stream, error) {
	stream, err := c.conn.CreateStream(headers, nil, false)
	if err != nil {
		return nil, err
	}
	if err = stream.WaitTimeout(5 * time.Second); err != nil {
		return nil, err
	}
	c.wg.Add(1)
	return &spdy31Stream{stream: stream, conn: c}, nil
}
