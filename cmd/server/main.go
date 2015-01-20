package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"

	"github.com/golang/glog"
	"github.com/kr/pty"
	"github.com/ncdc/httpstream/spdy"
)

func upgradeMe(w http.ResponseWriter, req *http.Request) {
	cmdBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		glog.Fatal(err)
	}
	cmdBuffer := bytes.NewBuffer(cmdBytes)
	cmdString := cmdBuffer.String()
	glog.Info(cmdString)
	cmdParts := strings.Split(cmdString, " ")

	streamer := spdy.NewResponseStreamer()
	glog.Info("calling StreamResponse")
	stdin, stdout, stderr, err := streamer.StreamResponse(w, req)
	if err != nil {
		glog.Fatalf("Unable to stream response %v", err)
	}

	glog.Info("creating command")
	command := exec.Command(cmdParts[0], cmdParts[1:]...)

	cp := func(s string, dst io.WriteCloser, src io.Reader) {
		defer func() {
			if s != "input" {
				dst.Close()
			}
		}()
		glog.V(1).Infof("START COPY %s", s)
		io.Copy(dst, src)
		glog.V(1).Infof("DONE COPY %s", s)
	}

	if req.Header.Get("TTY") != "1" {
		if stdin != nil {
			cmdIn, err := command.StdinPipe()
			if err != nil {
				glog.Fatal(err)
			}
			go func() {
				cp("input", cmdIn, stdin)
				// make sure we close the command's stdin when the stream is done
				cmdIn.Close()
			}()
		}

		if stdout != nil {
			cmdOut, err := command.StdoutPipe()
			if err != nil {
				glog.Fatal(err)
			}
			go cp("output", stdout, cmdOut)
		}

		if stderr != nil {
			cmdErr, err := command.StderrPipe()
			if err != nil {
				glog.Fatal(err)
			}
			go cp("error", stderr, cmdErr)
		}

		command.Run()
	} else {
		p, err := pty.Start(command)
		if err != nil {
			glog.Fatal(err)
		}
		if stdin != nil {
			go io.Copy(p, stdin)
		}
		if stdout != nil {
			go io.Copy(stdout, p)
		}
		command.Wait()
		// make sure to close the stdout stream!
		stdout.Close()
	}

	streamer.Wait()
}

func main() {
	flag.Parse()
	glog.Info("Listening")
	http.HandleFunc("/", upgradeMe)
	http.ListenAndServe("localhost:8888", nil)
}
