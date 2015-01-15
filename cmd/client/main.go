package main

import (
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/ncdc/httpstream/spdy"
)

func main() {
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

	var wg sync.WaitGroup
	cp := func(s string, dst io.Writer, src io.Reader) {
		defer func() {
			if s != "input" {
				if inputStream != nil {
					inputStream.Close()
				}
				wg.Done()
			}
		}()
		io.Copy(dst, src)
	}

	// stdin
	if *in {
		wg.Add(1)
		go func() {
			cp("input", inputStream, os.Stdin)
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
