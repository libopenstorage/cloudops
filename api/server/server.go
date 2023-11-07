package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Route is a specification and  handler for a REST endpoint.
type Route struct {
	verb string
	path string
	fn   func(http.ResponseWriter, *http.Request)
}

type restServer interface {
	Routes() []*Route
}

func (r *Route) GetVerb() string {
	return r.verb
}

func (r *Route) GetPath() string {
	return r.path
}

func (r *Route) GetFn() func(http.ResponseWriter, *http.Request) {
	return r.fn
}

func startOpsServer(port uint16) (*http.Server, error) {

}

func startServer(port uint16, rs restServer) (*http.Server, error) {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(notFound)
	for _, v := range rs.Routes() {
		router.Methods(v.verb).Path(v.path).HandlerFunc(v.fn)
	}
	if port != 0 {
		logrus.Printf("Starting REST service on port : %v", port)
		portServer := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: router}
		go portServer.ListenAndServe()
		return portServer, nil
	}
	// TODO: Implemet UDS
	return nil, fmt.Errorf("uds not supported")
}

func notFound(w http.ResponseWriter, r *http.Request) {
	logrus.Warnf("Not found: %+v ", r.URL)
	http.NotFound(w, r)
}
