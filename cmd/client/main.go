package main

import (
	"flag"
	"io"
	"net/http"
	"os"

	"github.com/golang/glog"
	"github.com/ncdc/httpstream"
	"github.com/ncdc/httpstream/api"
)

func main() {
	flag.Parse()

	req, err := http.NewRequest("POST", "http://localhost:8888/", nil)
	if err != nil {
		glog.Fatal(err)
	}
	upgrader := httpstream.NewRequestUpgrader()
	upgradedReq, err := upgrader.Upgrade(req, func(s api.Stream) {})
	if err != nil {
		glog.Fatal(err)
	}

	h := http.Header{}

	h.Set("type", "control")
	controlStream, err := upgradedReq.CreateStream(h)
	if err != nil {
		glog.Fatal(err)
	}

	h.Set("type", "input")
	inputStream, err := upgradedReq.CreateStream(h)
	if err != nil {
		glog.Fatal(err)
	}

	h.Set("type", "output")
	outputStream, err := upgradedReq.CreateStream(h)
	if err != nil {
		glog.Fatal(err)
	}

	h.Set("type", "error")
	errorStream, err := upgradedReq.CreateStream(h)
	if err != nil {
		glog.Fatal(err)
	}

	cp := func(s string, dst io.Writer, src io.Reader) {
		glog.Infof("Copying %s", s)
		io.Copy(dst, src)
		glog.Infof("DONE Copying %s", s)
	}

	go cp("input", inputStream, os.Stdin)
	go cp("output", os.Stdout, outputStream)
	go cp("error", os.Stderr, errorStream)

	b := make([]byte, 1)
	_, err = controlStream.Read(b)
	if err != nil && err != io.EOF {
		glog.Fatal(err)
	}

	errorStream.Close()
	outputStream.Close()
	inputStream.Close()
	controlStream.Close()
	upgradedReq.CloseWait()
	glog.Info("OVER")
}
