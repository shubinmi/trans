package trans

import (
	"net/http"

	gmux "github.com/gorilla/mux"
)

type HTTPRoute struct {
	Path         string
	EndpointOpts []HTTPOpt
	Method       string
}

func EnrichHTTP(mux *http.ServeMux, basePath string, routes []*HTTPRoute) {
	if basePath == "/" {
		basePath = ""
	}
	gm := gmux.NewRouter()
	serve := HTTP()
	for _, r := range routes {
		gm.HandleFunc(basePath+r.Path, serve(r.EndpointOpts...)).Methods(r.Method)
	}
	mux.Handle("/", gm)
}
