package spdy

import (
	"io"
	"net/http"

	"github.com/golang/glog"
	"github.com/ncdc/httpstream"
)

type spdy31RequestStreamer struct {
	conn httpstream.Connection
}

func NewRequestStreamer() httpstream.RequestStreamer {
	return &spdy31RequestStreamer{}
}

func (s *spdy31RequestStreamer) Stream3(req *http.Request, doStdin, doStdout, doStderr, tty bool) (stdin io.WriteCloser, stdout, stderr io.Reader, err error) {
	upgrader := NewRequestUpgrader()
	if tty {
		req.Header.Set("TTY", "1")
	}
	conn, err := upgrader.Upgrade(req, func(stream httpstream.Stream) {})
	if err != nil {
		return nil, nil, nil, err
	}

	h := http.Header{}

	if doStdin {
		h.Set("type", "input")
		stdin, err = conn.CreateStream(h)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if doStdout {
		h.Set("type", "output")
		stream, err := conn.CreateStream(h)
		if err != nil {
			return nil, nil, nil, err
		}
		// close our half of the output stream because we aren't writing to it
		stream.Close()
		stdout = stream
	}

	if doStderr && !tty {
		h.Set("type", "error")
		stream, err := conn.CreateStream(h)
		if err != nil {
			return nil, nil, nil, err
		}
		// close our half of the error stream because we aren't writing to it
		stream.Close()
		stderr = stream
	}

	s.conn = conn

	return stdin, stdout, stderr, nil
}

func (s *spdy31RequestStreamer) Wait() {
	glog.Info("Calling spdyConn.CloseWait()")
	s.conn.CloseWait()
	glog.Info("Conn closed")
}

func (s *spdy31ResponseStreamer) Wait() {
	glog.Info("Calling spdyConn.CloseWait()")
	s.conn.CloseWait()
	glog.Info("Conn closed")
}

type spdy31ResponseStreamer struct {
	inputStream  httpstream.Stream
	outputStream httpstream.Stream
	errorStream  httpstream.Stream
	ready        chan struct{}
	conn         httpstream.Connection
	tty          bool
}

func NewResponseStreamer() httpstream.ResponseStreamer {
	return &spdy31ResponseStreamer{
		ready: make(chan struct{}, 1),
	}
}

func (s *spdy31ResponseStreamer) StreamResponse(w http.ResponseWriter, req *http.Request) (stdin io.Reader, stdout, stderr io.WriteCloser, err error) {
	s.tty = req.Header.Get("TTY") == "1"
	upgrader := NewResponseUpgrader()
	conn, err := upgrader.Upgrade(w, req, s.newStreamHandler)
	if err != nil {
		return nil, nil, nil, err
	}
	<-s.ready
	s.conn = conn
	// close our half of the input stream, since we won't be writing to it
	s.inputStream.Close()
	return s.inputStream, s.outputStream, s.errorStream, nil
}

func (s *spdy31ResponseStreamer) newStreamHandler(stream httpstream.Stream) {
	typeString := stream.GetHeader("type")
	switch typeString {
	case "input":
		s.inputStream = stream
	case "output":
		s.outputStream = stream
	case "error":
		s.errorStream = stream
	}
	if s.inputStream != nil && s.outputStream != nil && (s.tty || s.errorStream != nil) {
		close(s.ready)
	}
}
