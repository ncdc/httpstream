package main

import (
	"flag"
	"io"
	"net"
	"net/http"
	"os/exec"

	"github.com/docker/spdystream"
	"github.com/golang/glog"
)

func main() {
	flag.Parse()
	glog.Info("Listening")
	listener, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		glog.Fatal(err)
	}

	for {
		rawConn, err := listener.Accept()
		if err != nil {
			glog.Fatal(err)
		}

		/*req, err := http.ReadRequest(rawConn)
		if err != nil {
			glog.Fatal(err)
		}*/

		session, err := newSession(rawConn, true)
		if err != nil {
			glog.Fatal(err)
		}
		session.Run()
		session.Close()
		rawConn.Close()
	}
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

func newSession(conn net.Conn, server bool) (*Session, error) {
	session := &Session{
		ready: make(chan struct{}, 1),
		stop:  make(chan struct{}, 1),
	}
	spdyConn, err := spdystream.NewConnection(conn, server)
	if err != nil {
		return nil, err
	}
	session.conn = spdyConn
	go spdyConn.Serve(session.newStreamHandler)
	return session, nil
}

func (s *Session) newStreamHandler(stream *spdystream.Stream) {
	typeString := stream.Headers().Get("type")
	returnHeaders := http.Header{}
	switch typeString {
	case "control":
		s.ControlStream = stream
	case "input":
		s.InputStream = stream
	case "output":
		s.OutputStream = stream
	case "error":
		s.ErrorStream = stream
	}
	if s.ControlStream != nil && s.InputStream != nil && s.OutputStream != nil && s.ErrorStream != nil {
		close(s.ready)
	}
	stream.SendReply(returnHeaders, false)
}

func (s *Session) Run() {
	<-s.ready
	command := exec.Command("bash")
	cp := func(s string, dst io.Writer, src io.Reader) {
		glog.Infof("Copying from %s", s)
		io.Copy(dst, src)
		glog.Infof("DONE Copying from %s", s)
	}
	if s.InputStream != nil {
		//command.Stdin = s.InputStream
		cmdIn, err := command.StdinPipe()
		if err != nil {
			glog.Fatal(err)
		}
		go cp("input", cmdIn, s.InputStream)
		// defer s.InputStream.Close()
	}
	if s.OutputStream != nil {
		// command.Stdout = s.OutputStream
		cmdOut, err := command.StdoutPipe()
		if err != nil {
			glog.Fatal(err)
		}
		go cp("output", s.OutputStream, cmdOut)
		// defer s.OutputStream.Close()
	}
	if s.ErrorStream != nil {
		// command.Stderr = s.ErrorStream
		cmdErr, err := command.StderrPipe()
		if err != nil {
			glog.Fatal(err)
		}
		go cp("error", s.ErrorStream, cmdErr)
		// defer s.ErrorStream.Close()
	}
	glog.Info("Running command")
	err := command.Start()
	if err != nil {
		glog.Infof("Error starting command: %v", err)
	}

	glog.Infof("%d", command.Process.Pid)

	err = command.Wait()
	if err != nil {
		glog.Infof("Error waiting command: %v", err)
	}
	controlHeaders := http.Header{}
	controlHeaders.Set("result", "ok")
	s.ControlStream.SendHeader(controlHeaders, true)
	glog.Info("DONE Run()")
}
