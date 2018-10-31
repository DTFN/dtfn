package httpserver

import (
	"net/http"
	"time"
	emtTypes "github.com/green-element-chain/gelchain/types"
)

type BaseServer struct {
	HttpServer *http.Server
}

func NewBaseServer(strategy *emtTypes.Strategy) *BaseServer {
	handler := NewTHandler(strategy)
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
