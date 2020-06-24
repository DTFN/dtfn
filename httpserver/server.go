package httpserver

import (
	"github.com/DTFN/dtfn/ethereum"
	"net/http"
	"time"
	emtTypes "github.com/DTFN/dtfn/types"
)

type BaseServer struct {
	HttpServer *http.Server
}

func NewBaseServer(strategy *emtTypes.Strategy,backend *ethereum.Backend) *BaseServer {
	handler := NewTHandler(strategy,backend)
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
