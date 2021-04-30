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
	"seckill/oauth-service/endpoints"
	"seckill/oauth-service/service"
)

var (
	ErrorTokenRequest = errors.New("invalid request token")
	ErrInvalidClientRequest =  errors.New("invalid request client")
	ErrCheckTokenRequest =  errors.New("invalid request check token")
)
func errorEncoder(_ context.Context, err error, w http.ResponseWriter){
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err{
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func decodeTokenRequest(ctx context.Context, req *http.Request) (request interface{}, err error) {
grantType := req.URL.Query().Get("grantType")
if grantType == ""{
return nil, ErrorTokenRequest
}
return &endpoints.TokenRequest{
GrantType: grantType,
Reader: req,
}, nil
}

func encodeJsonResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error{
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

func decodeCheckTokenRequest(ctx context.Context, req *http.Request) (request interface{}, err error) {
	tokenValue := req.URL.Query().Get("token")
	if tokenValue == ""{
		return nil, ErrCheckTokenRequest
	}
	return endpoints.CheckTokenRequest{
		TokenValue: tokenValue,
	}, nil
}

func MakeHttpHandler(ctx context.Context, endpts endpoints.OauthEndpoints,clientSvc service.ClientDetailsService,logger log.Logger) http.Handler{
	r := mux.NewRouter()
	options := []kitHttp.ServerOption{
		kitHttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kitHttp.ServerErrorEncoder(errorEncoder),
	}

	clientAuthorizationOpts := []kitHttp.ServerOption{
		kitHttp.ServerBefore(makeClientAuthorizationContext(clientSvc, logger)),
		kitHttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kitHttp.ServerErrorEncoder(errorEncoder),
	}

	r.Methods("GET").Path("/health").Handler(kitHttp.NewServer(
		endpts.HealthEndpoint,
		kitHttp.NopRequestDecoder,
		encodeJsonResponse,
		options...,
	))

	r.Methods("POST").Path("/oauth/token").Handler(kitHttp.NewServer(
		endpts.TokenEndpoint,
		decodeTokenRequest,
		encodeJsonResponse,
		clientAuthorizationOpts...,
	))
	r.Methods("POST").Path("/oauth/check_token").Handler(kitHttp.NewServer(
		endpts.CheckTokenEndpoint,
		decodeCheckTokenRequest,
		encodeJsonResponse,
		clientAuthorizationOpts...,
	))
	return r
}

func makeClientAuthorizationContext(clientDetailsSvc service.ClientDetailsService, logger log.Logger) kitHttp.RequestFunc{
	return func(ctx context.Context, r *http.Request) context.Context {
		if clientId, clientSecret, ok := r.BasicAuth(); ok{
			clientDetails, err := clientDetailsSvc.GetClientDetailByClientId(ctx, clientId, clientSecret)
			if err == nil{
				return context.WithValue(ctx, endpoints.OAuth2ClientDetailsKey, clientDetails)
			}
		}
		return context.WithValue(ctx, endpoints.OAuth2ErrorKey, ErrInvalidClientRequest)
	}
}
