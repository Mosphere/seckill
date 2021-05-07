package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"seckill/pkg/discover"
	"seckill/pkg/loadbalance"
	"seckill/string-service/config"
	"strconv"

	"github.com/afex/hystrix-go/hystrix"
)

const (
	StringSvcCmdName = "String.string"
	StringSvc        = "string"
)

// Service Define a service interface
type Service interface {
	// 远程调用 string-service 服务
	UseStringService(operationType, a, b string) (string, error)

	// 健康检查
	HealthCheck() bool
}

type UseStringService struct {
	client      *discover.DiscoveryClient
	loadBalance loadbalance.LoadBalance
}

// StringResponse define response struct
type StringResponse struct {
	Result string `json:"result"`
	Error  error  `json:"error"`
}

func NewService(client *discover.DiscoveryClient, lb loadbalance.LoadBalance) *UseStringService {
	// 设置触发最低请求阀值为 5，方便我们观察结果
	hystrix.ConfigureCommand(StringSvcCmdName, hystrix.CommandConfig{
		RequestVolumeThreshold: 5,
	})
	return &UseStringService{
		client:      client,
		loadBalance: lb,
	}
}

func (s *UseStringService) UseStringService(opType, a, b string) (string, error) {
	var opResult string
	instances := s.client.DiscoveryServices(StringSvc, config.Logger)
	selectInstance, err := s.loadBalance.SelectService(instances)
	if err == nil {
		requestUrl := url.URL{
			Scheme: "http",
			Host:   selectInstance.Host + ":" + strconv.Itoa(selectInstance.Port),
			Path:   fmt.Sprintf("/op/%s/%s/%s", opType, a, b),
		}
		config.Logger.Printf("current string-service ID is %s and address:port is %s:%d\n", selectInstance.ID, selectInstance.Host, selectInstance.Port)
		resp, err := http.Post(requestUrl.String(), "", nil)
		if err == nil {
			result := &StringResponse{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			if err == nil && result.Error == nil {
				opResult = result.Result
			}
		}
	}
	return opResult, err
}

func (s *UseStringService) HealthCheck() bool {
	return true
}
