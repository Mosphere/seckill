package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"seckill/oauth-service/endpoints"
	"seckill/oauth-service/service"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	kitHttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/longjoy/micro-go-book/ch11-security/endpoint"
)

var (
	ErrorTokenRequest       = errors.New("invalid request token")
	ErrInvalidClientRequest = errors.New("invalid request client")
	ErrCheckTokenRequest    = errors.New("invalid request check token")
)

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err {
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func decodeTokenRequest(ctx context.Context, req *http.Request) (request interface{}, err error) {
	grantType := req.URL.Query().Get("grantType")
	if grantType == "" {
		return nil, ErrorTokenRequest
	}
	return &endpoints.TokenRequest{
		GrantType: grantType,
		Reader:    req,
	}, nil
}

func encodeJsonResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

func decodeCheckTokenRequest(ctx context.Context, req *http.Request) (request interface{}, err error) {
	tokenValue := req.URL.Query().Get("token")
	if tokenValue == "" {
		return nil, ErrCheckTokenRequest
	}
	return endpoints.CheckTokenRequest{
		TokenValue: tokenValue,
	}, nil
}

func decodeSimpleRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return &endpoint.SimpleRequest{}, nil
}

func MakeHttpHandler(ctx context.Context, endpts endpoints.OauthEndpoints, tokenSvc service.TokenService, clientSvc service.ClientDetailsService, logger log.Logger) http.Handler {
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

	oAuth2AuthorizationOpts := []kitHttp.ServerOption{
		kitHttp.ServerBefore(makeOAuth2AuthorizationContext(tokenSvc, logger)),
		kitHttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kitHttp.ServerErrorEncoder(errorEncoder),
	}

	r.Methods("Get").Path("/simple").Handler(kitHttp.NewServer(
		endpts.SimpleEndpoint,
		decodeSimpleRequest,
		encodeJsonResponse,
		oAuth2AuthorizationOpts...,
	))
	return r
}

//构建认证上下文验证器
func makeOAuth2AuthorizationContext(tokenSvc service.TokenService, logger log.Logger) kitHttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		accessTokenValue := r.Header.Get("Authorization")
		if accessTokenValue != "" {
			//获取令牌对应的客户端信息
			oauth2Details, err := tokenSvc.GetOAuth2DetailsByAccessToken(accessTokenValue)
			if err != nil {
				return context.WithValue(ctx, endpoints.OAuth2ErrorKey, ErrorTokenRequest)
			}
			return context.WithValue(ctx, endpoints.OAuth2DetailsKey, oauth2Details)
		}
		return context.WithValue(ctx, endpoints.OAuth2ErrorKey, ErrorTokenRequest)
	}
}

func makeClientAuthorizationContext(clientDetailsSvc service.ClientDetailsService, logger log.Logger) kitHttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		if clientId, clientSecret, ok := r.BasicAuth(); ok {
			clientDetails, err := clientDetailsSvc.GetClientDetailByClientId(ctx, clientId, clientSecret)
			if err == nil {
				return context.WithValue(ctx, endpoints.OAuth2ClientDetailsKey, clientDetails)
			}
		}
		return context.WithValue(ctx, endpoints.OAuth2ErrorKey, ErrInvalidClientRequest)
	}
}
