package spdy

import (
	"sync"

	"github.com/docker/spdystream"
)

type spdy31Stream struct {
	stream *spdystream.Stream
	conn   *spdy31Connection
	closed bool
	lock   sync.Mutex
}

func (s *spdy31Stream) GetHeader(key string) string {
	return s.stream.Headers().Get(key)
}

func (s *spdy31Stream) Read(p []byte) (n int, err error) {
	return s.stream.Read(p)
}

func (s *spdy31Stream) Write(data []byte) (n int, err error) {
	return s.stream.Write(data)
}

func (s *spdy31Stream) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	defer s.conn.wg.Done()
	return s.stream.Close()
}
