package endpoint

import (
	"context"
	"github.com/go-kit/kit/endpoint"
	"seckill/use-string-service/service"
)

// CalculateEndpoint define endpoint
type UseStringEndpoints struct {
	UseStringEndpoint      endpoint.Endpoint
	HealthCheckEndpoint endpoint.Endpoint
}


// StringRequest define request struct
type UseStringRequest struct {
	RequestType string `json:"request_type"`
	A           string `json:"a"`
	B           string `json:"b"`
}

// StringResponse define response struct
type UseStringResponse struct {
	Result string `json:"result"`
	Error  string  `json:"error"`
}

//负责处理
func MakeUseStringService(svc service.Service) endpoint.Endpoint{
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req, _ := request.(UseStringRequest)
		var (
			res, a, b string
			opError   error
		)
		a = req.A
		b = req.B
		op := req.RequestType
		res, opError = svc.UseStringService(op,a, b)
		return UseStringResponse{Result: res}, opError
	}
}

// HealthRequest 健康检查请求结构
type HealthRequest struct{}

// HealthResponse 健康检查响应结构
type HealthResponse struct {
	Status bool `json:"status"`
}

// MakeHealthCheckEndpoint 创建健康检查Endpoint
func MakeHealthCheckEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		status := svc.HealthCheck()
		return HealthResponse{status}, nil
	}
}