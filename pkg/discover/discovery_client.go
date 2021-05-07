package discover

import (
	"log"
	"seckill/pkg/common"
	"sync"

	"github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
)

type DiscoveryClient struct {
	Host   string
	Port   int
	config *api.Config
	client consul.Client
	mutex  *sync.Mutex
	//服务实例缓存字段
	instancesMap sync.Map
}

type IDiscoveryClient interface {
	/**
		 * 服务注册接口
	     * @Description:
	     * @param instanceId 服务实例Id
	     * @param svcHost
	     * @param healthCheckUrl
	     * @param svcPort
	     * @param svcName
	     * @param weight 权重
	     * @param meta 服务实例元数据
	     * @param tags
	     * @param logger
	     * @return bool
	*/
	Register(instanceId, svcHost, healthCheckUrl, svcPort, svcName string, weight int, meta map[string]string, tags []string, logger *log.Logger) bool

	/**
		 * 服务注销接口
	     * @Description:
	     * @param instanceId 服务实例Id
	     * @param logger
	     * @return bool
	*/
	DeRegister(instanceId string, logger *log.Logger) bool

	DiscoveryServices(svcName string, logger *log.Logger) []*common.ServiceInstance
}
