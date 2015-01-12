package main

import (
	"flag"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/docker/spdystream"
	"github.com/openshift/origin/Godeps/_workspace/src/github.com/golang/glog"

	_ "net/http/pprof"
)

func main() {
	flag.Parse()

	go func() {
		http.ListenAndServe("localhost:9999", nil)
	}()

	//var req http.Request
	//var resp http.Response
	var conn net.Conn
	glog.Info("Dialing")

	conn, err := net.Dial("tcp", "localhost:8888")
	if err != nil {
		glog.Fatal(err)
	}
	defer conn.Close()

	// write request, including upgrade headers
	//req.Write(conn)

	// get response, which should say we're upgraded
	/*resp, err = http.ReadResponse(conn, req)
	if err != nil {
		glog.Fatalf(err)
	}*/

	// check upgrade response status

	session, err := newSession(conn, false)
	if err != nil {
		glog.Fatal(err)
	}
	defer session.Close()
	session.Run()
	session.Close()
}

type Session struct {
	ControlStream *spdystream.Stream
	InputStream   *spdystream.Stream
	OutputStream  *spdystream.Stream
	ErrorStream   *spdystream.Stream
	ready         chan struct{}
	stop          chan struct{}
	conn          *spdystream.Connection
}

func (s *Session) Close() {

	if s.ControlStream != nil {
		s.ControlStream.Close()
	}
	if s.InputStream != nil {
		s.InputStream.Close()
	}
	if s.OutputStream != nil {
		s.OutputStream.Close()
	}
	if s.ErrorStream != nil {
		s.ErrorStream.Close()
	}
	s.conn.CloseWait()
}

func (s *Session) createStream(streamType string) (*spdystream.Stream, error) {
	h := http.Header{}
	h.Set("type", streamType)
	glog.Infof("Creating stream %#v", h)
	return s.conn.CreateStream(h, nil, false)
}

func newSession(conn net.Conn, server bool) (*Session, error) {
	session := &Session{
		stop: make(chan struct{}, 3),
	}
	cp := func(s string, dst io.Writer, src io.Reader, c io.Closer) {
		glog.Infof("Copying from %s", s)
		io.Copy(dst, src)
		glog.Infof("DONE Copying from %s", s)
		//c.Close()
		//session.stop <- struct{}{}
	}
	spdyConn, err := spdystream.NewConnection(conn, server)
	if err != nil {
		return nil, err
	}
	session.conn = spdyConn
	go spdyConn.Serve(spdystream.NoOpStreamHandler)

	session.ControlStream, err = session.createStream("control")
	if err != nil {
		glog.Fatal(err)
	}

	session.InputStream, err = session.createStream("input")
	if err != nil {
		glog.Fatal(err)
	}
	go cp("input", session.InputStream, os.Stdin, session.InputStream)

	session.OutputStream, err = session.createStream("output")
	if err != nil {
		glog.Fatal(err)
	}
	go cp("output", os.Stdout, session.OutputStream, session.OutputStream)

	session.ErrorStream, err = session.createStream("error")
	if err != nil {
		glog.Fatal(err)
	}
	go cp("error", os.Stderr, session.ErrorStream, session.ErrorStream)
	return session, nil
}

func (s *Session) Run() {
	// closech := make(chan *spdystream.Stream)
	// s.conn.NotifyClose(closech, time.Duration(0))
	controlHeaders, err := s.ControlStream.ReceiveHeader()
	if err != nil {
		glog.Fatal(err)
	}
	if controlHeaders.Get("result") == "ok" {
		glog.Info("result ok")
	}
	// select {
	// case <-closech:
	// }

	glog.Info("DONE RUNNING")
}
