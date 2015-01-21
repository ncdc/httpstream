package main

import (
	"bytes"
	"flag"
	"io"
	"net/http"
	"os"

	"github.com/docker/spdystream"
	"github.com/golang/glog"
)

type foo struct {
	bytes.Buffer
}

func (f foo) Close() error {
	glog.Infof("foo close")
	return nil
}

func main() {
	flag.Parse()
	client := &http.Client{}

	body := &foo{}
	conn, err := spdystream.NewConnection(body, false)
	if err != nil {
		glog.Fatalf("spdy newconn err: %v", err)
	}
	req, err := http.NewRequest("POST", "http://localhost:6666", body)
	if err != nil {
		glog.Fatalf("new req err: %v", err)
	}
	resp, err := client.Do(req)
	_ = resp
	if err != nil {
		glog.Fatalf("resp err: %v", err)
	}
	/*
			dial, err := net.Dial("tcp", "localhost:6666")
			if err != nil {
				glog.Fatalf("dial err: %v", err)
			}
			req, err := http.NewRequest("POST", "http://localhost:6666", nil)
			if err != nil {
				glog.Fatalf("new req err: %v", err)
			}
			req.Write(dial)
			http.ReadResponse(bufio.NewReader(dial), req)
		conn, err := spdystream.NewConnection(dial, false)
		if err != nil {
			glog.Fatalf("spdy newconn err: %v", err)
		}
	*/
	go conn.Serve(spdystream.NoOpStreamHandler)
	stream, err := conn.CreateStream(http.Header{}, nil, false)
	if err != nil {
		glog.Fatalf("create stream err: %v", err)
	}
	go io.Copy(os.Stdout, stream)
	go io.Copy(stream, os.Stdin)
	select {}
}
