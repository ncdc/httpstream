package main

import (
	"flag"
	"net/http"

	"github.com/docker/spdystream"
	"github.com/golang/glog"
)

func upgradeMe(w http.ResponseWriter, req *http.Request) {
	glog.Infof("writing header")
	w.WriteHeader(http.StatusSwitchingProtocols)
	glog.Infof("written, hijacking")
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		glog.Fatalf("hijack err: %v", err)
	}
	//defer conn.Close()

	glog.Infof("new spdy conn")
	sconn, err := spdystream.NewConnection(conn, true)
	if err != nil {
		glog.Fatalf("spdy newconn err: %v", err)
	}

	glog.Infof("starting serve")
	go sconn.Serve(spdystream.MirrorStreamHandler)
	glog.Infof("select")
	select {}
}

func main() {
	flag.Parse()
	glog.Info("Listening")
	http.HandleFunc("/", upgradeMe)
	http.ListenAndServe("localhost:6666", nil)
}
