package spdy

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/docker/spdystream"
	"github.com/ncdc/httpstream"
)

const (
	headerConnection = "Connection"
	headerUpgrade    = "Upgrade"
	headerSpdy31     = "SPDY/3.1"
)

type spdy31RequestUpgrader struct {
}

func NewRequestUpgrader() httpstream.RequestUpgrader {
	return spdy31RequestUpgrader{}
}

func (u spdy31RequestUpgrader) Upgrade(req *http.Request, newStreamHandler httpstream.NewStreamHandler) (httpstream.Connection, error) {
	req.Header.Add(headerConnection, headerUpgrade)
	req.Header.Add(headerUpgrade, headerSpdy31)

	conn, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		return nil, err
	}

	err = req.Write(conn)
	if err != nil {
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, err
	}

	if !headerMatch(req.Header, headerConnection, headerUpgrade) || !headerMatch(req.Header, headerUpgrade, headerSpdy31) {
		return nil, fmt.Errorf("Expected upgrade to SPDY/3.1 from server; got %#v instead", resp.Header)
	}

	spdyConn, err := spdystream.NewConnection(conn, false)
	if err != nil {
		return nil, err
	}

	c := &spdy31Connection{conn: spdyConn}

	go spdyConn.Serve(func(s *spdystream.Stream) {
		newStreamHandler(&spdy31Stream{stream: s, conn: c})
		s.SendReply(http.Header{}, false)
	})

	return c, nil
}

func NewResponseUpgrader() httpstream.ResponseUpgrader {
	return spdy31ResponseUpgrader{}
}

type spdy31ResponseUpgrader struct{}

func headerMatch(h http.Header, key, value string) bool {
	found := false
	headers := h[key]
	if len(headers) > 0 {
		for _, header := range headers {
			if strings.ToLower(header) == strings.ToLower(value) {
				found = true
				break
			}
		}
	}
	return found
}

func (u spdy31ResponseUpgrader) Upgrade(w http.ResponseWriter, req *http.Request, newStreamHandler httpstream.NewStreamHandler) (httpstream.Connection, error) {
	if !headerMatch(req.Header, headerConnection, headerUpgrade) || !headerMatch(req.Header, headerUpgrade, headerSpdy31) {
		return nil, fmt.Errorf("Missing upgrade headers: %v", req.Header)
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("w is not a hijacker: %v", w)
	}

	w.Header().Add(headerConnection, headerUpgrade)
	w.Header().Add(headerUpgrade, headerSpdy31)
	w.WriteHeader(http.StatusSwitchingProtocols)

	conn, _, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}

	spdyConn, err := spdystream.NewConnection(conn, true)
	if err != nil {
		defer conn.Close()
		return nil, err
	}

	c := &spdy31Connection{conn: spdyConn}

	go spdyConn.Serve(func(s *spdystream.Stream) {
		c.wg.Add(1)
		newStreamHandler(&spdy31Stream{stream: s, conn: c})
		s.SendReply(http.Header{}, false)
	})

	return c, nil
}
