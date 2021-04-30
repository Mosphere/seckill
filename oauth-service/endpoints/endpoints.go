package endpoints

import (
	"context"
	"errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"net/http"
	"seckill/oauth-service/model"
	"seckill/oauth-service/service"
)

const (
	OAuth2DetailsKey       = "OAuth2Details"
	OAuth2ClientDetailsKey = "OAuth2ClientDetails"
	OAuth2ErrorKey         = "OAuth2Error"
)

var (
	ErrInvalidClientRequest = errors.New("invalid client message")
)


type OauthEndpoints struct{
	TokenEndpoint endpoint.Endpoint
	CheckTokenEndpoint endpoint.Endpoint
	HealthEndpoint endpoint.Endpoint
}

type TokenRequest struct {
	GrantType string
	Reader *http.Request
}
type TokenResponse struct {
	AccessToken *model.OAuth2Token `json:"access_token"`
	Error string `json:"error"`
}
func MakeTokenEndpoint(svc service.TokenGranter) endpoint.Endpoint{
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req, ok := request.(*TokenRequest)
		if !ok{
			return nil, ErrInvalidClientRequest
		}
		token, err := svc.Grant(ctx, req.GrantType, ctx.Value(OAuth2ClientDetailsKey).(*model.ClientDetails), req.Reader)
		var errString = ""
		if err != nil{
			errString = err.Error()
		}

		return TokenResponse{
			AccessToken: token,
			Error: errString,
		}, nil
	}
}

func MakeClientAuthorizationMiddleware(logger log.Logger) endpoint.Middleware{
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if err, ok := ctx.Value(OAuth2DetailsKey).(error); ok{
				return nil, err
			}
			if _, ok := ctx.Value(OAuth2ClientDetailsKey).(*model.OAuth2Details); !ok{
				return nil, ErrInvalidClientRequest
			}
			return next(ctx, request)
		}
	}
}
type CheckTokenRequest struct {
	TokenValue string
	ClientDetails *model.ClientDetails
}

type CheckTokenResponse struct {
	OAuth2Details *model.OAuth2Details
	Error string
}
func MakeCheckTokenEndpoint(svc service.TokenService) endpoint.Endpoint{
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req, ok := request.(CheckTokenRequest)
		if !ok{
			return nil, ErrInvalidClientRequest
		}
		oauth2Details, err := svc.GetOAuth2DetailsByAccessToken(req.TokenValue)
		var errString = ""
		if err != nil{
			errString = err.Error()
		}
		return CheckTokenResponse{
			OAuth2Details: oauth2Details,
			Error: errString,
		}, err
	}
}

type HealthResponse struct {
	Status bool `json:"status"`
}

func MakeHealthEndpoint(svc service.Service) endpoint.Endpoint{
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		return HealthResponse{
			Status: svc.Health(),
		}, nil
	}
}