package httpserver

import (
	"net/http"
	"time"
)

type BaseServer struct {
	HttpServer *http.Server
}

func NewBaseServer() *BaseServer {
	handler := NewTHandler()
	handler.RegisterFunc()
	return &BaseServer{
		HttpServer: &http.Server{
			Addr:           ":19190",
			Handler:        handler,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
	}
}