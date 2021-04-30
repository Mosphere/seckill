package transport

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	kitHttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"net/http"
	"seckill/string-service/endpoint"
)

var (
	ErrorBadRequest = errors.New("invalid request parameter")
)

func errorEncoder(ctx context.Context, ){

}
// MakeHttpHandler make http handler use mux
func MakeHttpHandler(ctx context.Context, endpoints endpoint.StringEndpoints, logger log.Logger) http.Handler {
	r := mux.NewRouter()

	options := []kitHttp.ServerOption{
		kitHttp.ServerBefore(),
		kitHttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kitHttp.ServerErrorEncoder(kitHttp.DefaultErrorEncoder),
	}

	r.Methods("POST").Path("/op/{type}/{a}/{b}").Handler(kitHttp.NewServer(
		endpoints.StringEndpoint,
		decodeStringRequest,
		encodeStringResponse,
		options...,
	))

	//r.Path("/metrics").Handler(promhttp.Handler())

	// create health check handler
	r.Methods("GET").Path("/health").Handler(kitHttp.NewServer(
		endpoints.HealthCheckEndpoint,
		decodeHealthCheckRequest,
		encodeStringResponse,
		options...,
	))

	return r
}

// decodeStringRequest decode request params to struct
func decodeStringRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	requestType, ok := vars["type"]
	if !ok {
		return nil, ErrorBadRequest
	}

	pa, ok := vars["a"]
	if !ok {
		return nil, ErrorBadRequest
	}

	pb, ok := vars["b"]
	if !ok {
		return nil, ErrorBadRequest
	}

	return endpoint.StringRequest{
		RequestType: requestType,
		A:           pa,
		B:           pb,
	}, nil
}

// encodeStringResponse encode response to return
func encodeStringResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

// decodeHealthCheckRequest decode request
func decodeHealthCheckRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return endpoint.HealthRequest{}, nil
}
