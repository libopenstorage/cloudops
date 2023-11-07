package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/libopenstorage/openstorage/api"
)

type ops struct{}

func (o *ops) Routes() []*Route {
	routes := []*Route{
		o.attachRoute(),
		o.detachRoute(),
		o.deleteRoute()}
	return routes
}

func (o *ops) attachRoute() *Route {
	return &Route{verb: "PUT", path: volPath("/{id}", ops.APIVersion), fn: o.volumeSet}
}

func (o *ops) detachRoute() *Route {
	return &Route{verb: "PUT", path: volPath("/{id}", ops.APIVersion), fn: o.volumeSet}
}

func (o *ops) deleteRoute() *Route {
	return &Route{verb: "PUT", path: volPath("/{id}", ops.APIVersion), fn: o.volumeSet}
}

// Creates a single volume with given spec.
func (o *ops) create(w http.ResponseWriter, r *http.Request) {
	var dcRes api.VolumeCreateResponse
	var dcReq api.VolumeCreateRequest
	method := "create"

	if err := json.NewDecoder(r.Body).Decode(&dcReq); err != nil {
		fmt.Println("returning error here")
		o.sendError(o.name, method, w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get context with auth token
	ctx, cancel, err := o.annotateContext(r)
	defer cancel()
	if err != nil {
		o.sendError(o.name, method, w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get gRPC connection
	conn, err := o.getConn()
	if err != nil {
		o.sendError(o.name, method, w, err.Error(), http.StatusInternalServerError)
		return
	}
	spec := dcReq.GetSpec()
	if spec.VolumeLabels == nil {
		spec.VolumeLabels = make(map[string]string)
	}
	for k, v := range dcReq.Locator.GetVolumeLabels() {
		spec.VolumeLabels[k] = v
	}

	volumes := api.NewOpenStorageVolumeClient(conn)
	id, err := volumes.Create(ctx, &api.SdkVolumeCreateRequest{
		Name:   dcReq.Locator.GetName(),
		Labels: dcReq.Locator.GetVolumeLabels(),
		Spec:   dcReq.GetSpec(),
	})

	dcRes.VolumeResponse = &api.VolumeResponse{Error: responseStatus(err)}
	if err == nil {
		dcRes.Id = id.GetVolumeId()
	}

	json.NewEncoder(w).Encode(&dcRes)
}
