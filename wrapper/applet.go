package wrapper

import "net/http"

type Client struct {
}

func NewServer(name string, host string, port int, handlers map[string]http.HandlerFunc) *Server {

}

type Server struct {
	*http.ServeMux
	name string
}

func (svr *Server) Start() {

}
