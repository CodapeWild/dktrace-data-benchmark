package wrapper

import "net/http"

type Client struct {
}

type Server struct {
	name string
}

func (svr *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

}
