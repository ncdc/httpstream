package main

import (
	"flag"
	"io"
	"net/http"
	"os/exec"

	"github.com/golang/glog"
	"github.com/ncdc/httpstream"
	"github.com/ncdc/httpstream/spdy"
)

type ExecHandler struct {
	ControlStream httpstream.Stream
	InputStream   httpstream.Stream
	OutputStream  httpstream.Stream
	ErrorStream   httpstream.Stream

	conn  httpstream.Connection
	ready chan struct{}
}

func (h *ExecHandler) newStreamHandler(stream httpstream.Stream) {
	typeString := stream.GetHeader("type")
	switch typeString {
	case "control":
		h.ControlStream = stream
	case "input":
		h.InputStream = stream
	case "output":
		h.OutputStream = stream
	case "error":
		h.ErrorStream = stream
	}

	if h.ControlStream != nil && h.InputStream != nil && h.OutputStream != nil && h.ErrorStream != nil {
		close(h.ready)
	}
}

func (h *ExecHandler) Run() {
	<-h.ready

	command := exec.Command("bash")
	cp := func(s string, dst io.Writer, src io.Reader) {
		glog.Infof("Copying %s", s)
		io.Copy(dst, src)
		glog.Infof("DONE Copying %s", s)
	}
	if h.InputStream != nil {
		cmdIn, err := command.StdinPipe()
		if err != nil {
			glog.Fatal(err)
		}
		go cp("input", cmdIn, h.InputStream)
	}
	if h.OutputStream != nil {
		cmdOut, err := command.StdoutPipe()
		if err != nil {
			glog.Fatal(err)
		}
		go cp("output", h.OutputStream, cmdOut)
	}
	if h.ErrorStream != nil {
		cmdErr, err := command.StderrPipe()
		if err != nil {
			glog.Fatal(err)
		}
		go cp("error", h.ErrorStream, cmdErr)
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
	h.ControlStream.Close()
	h.InputStream.Close()
	h.OutputStream.Close()
	h.ErrorStream.Close()
	glog.Info("DONE Run()")
}

func upgradeMe(w http.ResponseWriter, req *http.Request) {
	upgrader := spdy.NewResponseUpgrader()
	h := &ExecHandler{
		ready: make(chan struct{}, 1),
	}
	var err error
	h.conn, err = upgrader.Upgrade(w, req, h.newStreamHandler)
	if err != nil {
		glog.Error(err)
	}
	h.Run()
	h.conn.CloseWait()
}

func main() {
	flag.Parse()
	glog.Info("Listening")
	http.HandleFunc("/", upgradeMe)
	http.ListenAndServe("0.0.0.0:8888", nil)
}
