package httpserver

import (
	"io"
	"net/http"
)

type THandler struct {
	HandlersMap map[string]HandlersFunc
}

type HandlersFunc func(http.ResponseWriter, *http.Request)

func NewTHandler() *THandler {
	return &THandler{
		HandlersMap: make(map[string]HandlersFunc),
	}
}

func (tHandler *THandler) RegisterFunc() {
	tHandler.HandlersMap["/hello"] = Hello
	tHandler.HandlersMap["/f1"] = f1
	tHandler.HandlersMap["/f2"] = f2
}

func (tHandler *THandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := tHandler.HandlersMap[r.URL.String()]; ok {
		h(w, r)
	}
}

func f1(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "111111111111")
}

func f2(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "2222222222222")
}

func Hello(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Hello"))
}