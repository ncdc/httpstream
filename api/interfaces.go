package api

import (
	"io"
	"net/http"
)

type NewStreamHandler func(Stream)

type RequestUpgrader interface {
	Upgrade(request *http.Request, newStreamHandler NewStreamHandler) (Connection, error)
}

type Connection interface {
	io.Closer
	CloseWait() error
	CreateStream(headers http.Header) (Stream, error)
}

type Stream interface {
	io.ReadWriteCloser
	GetHeader(key string) string
}

type ResponseUpgrader interface {
	Upgrade(w http.ResponseWriter, req *http.Request, newStreamHandler NewStreamHandler) (Connection, error)
}
