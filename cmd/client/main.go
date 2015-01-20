package main

import (
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/docker/docker/pkg/term"
	"github.com/golang/glog"
	"github.com/ncdc/httpstream/spdy"

	_ "net/http/pprof"
)

func main() {
	go func() {
		http.ListenAndServe("localhost:7777", nil)
	}()
	tty := flag.Bool("t", false, "tty")
	in := flag.Bool("i", false, "in")
	flag.Parse()

	args := strings.Join(flag.Args(), " ")
	cmdReader := strings.NewReader(args)
	req, err := http.NewRequest("POST", "http://localhost:8888/", cmdReader)
	if err != nil {
		glog.Fatal(err)
	}

	requestStreamer := spdy.NewRequestStreamer()
	inputStream, outputStream, errorStream, err := requestStreamer.Stream3(req, *in, true, true, *tty)
	if err != nil {
		glog.Fatal(err)
	}

	var inFd uintptr
	isTerminalIn := false
	if *in {
		inFd = os.Stdin.Fd()
		isTerminalIn = term.IsTerminal(inFd)
	}

	r, w := io.Pipe()
	go func() {
		io.Copy(w, os.Stdin)
		r.Close()
	}()
	var wg sync.WaitGroup
	cp := func(s string, dst io.Writer, src io.Reader) {
		defer func() {
			if s != "input" {
				if inputStream != nil {
					inputStream.Close()
					r.Close()
					w.Close()
				}
				wg.Done()
			}
		}()
		glog.Infof("START COPY %s", s)
		io.Copy(dst, src)
		glog.Infof("DONE COPY %s", s)
	}

	// stdin
	if *in {
		if isTerminalIn && *tty {
			oldState, err := term.SetRawTerminal(inFd)
			if err != nil {
				glog.Fatal(err)
			}
			defer term.RestoreTerminal(inFd, oldState)
		}
		wg.Add(1)
		go func() {
			cp("input", inputStream, r)
			inputStream.Close()
			wg.Done()
		}()
	}

	// stdout
	wg.Add(1)
	go cp("output", os.Stdout, outputStream)

	// stderr
	if !*tty && errorStream != nil {
		wg.Add(1)
		go cp("error", os.Stderr, errorStream)
	}

	glog.Infof("wg wait")
	wg.Wait()
	glog.Infof("rs wait")
	requestStreamer.Wait()
}
