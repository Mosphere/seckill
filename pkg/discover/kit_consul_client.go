package discover

import (
	"github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"

	"log"
	"seckill/pkg/common"
	"strconv"
)

/**
注册服务
 */
func (consulClt *DiscoveryClient)Register(instanceId, svcHost,svcPort, healthCheckUrl, svcName string, weight int, meta map[string]string, tags []string, logger *log.Logger) bool{
	port, _ := strconv.Atoi(svcPort)
	serviceRegistration := &api.AgentServiceRegistration{
		ID: instanceId,
		Name: svcName,
		Tags: tags,
		Address: svcHost,
		Port: port,
		Meta: meta,
		Weights: &api.AgentWeights{
			Passing: weight,
		},
		Check: &api.AgentServiceCheck{
			DeregisterCriticalServiceAfter: "30s",
			HTTP: "http://" + svcHost + ":" + strconv.Itoa(port) + healthCheckUrl,
			Interval: "15s",

		},
	}

	err := consulClt.client.Register(serviceRegistration)
	if err != nil{
		if logger != nil{
			logger.Println("Register Service Error!")
		}
		return false
	}
	//构建服务实例
	if logger != nil{
		logger.Println("Register Service Success!")
	}
	return true
}

func New(consulHost , consulPort string) *DiscoveryClient{
	port, _ := strconv.Atoi(consulPort)
	apiConfig := api.DefaultConfig()
	apiConfig.Address = consulHost + ":" + strconv.Itoa(port)
	apiClient, err := api.NewClient(apiConfig)
	if err != nil{
		return nil
	}
	consulClt := consul.NewClient(apiClient)
	return &DiscoveryClient{
		Host: consulHost,
		Port: port,
		client: consulClt,
		config: apiConfig,
	}
}

func (consulClt *DiscoveryClient)DeRegister(instanceId string, logger *log.Logger) bool{
	serviceRegistration := &api.AgentServiceRegistration{
		ID: instanceId,
	}
	err := consulClt.client.Deregister(serviceRegistration)
	if err != nil {
		if logger != nil {
			logger.Println("Deregister Service Error!")
		}
		return false
	}
	if logger != nil {
		logger.Println("Deregister Service Success!")
	}
	return true
}
/*
发现服务
 */
func (consulClt *DiscoveryClient)DiscoveryServices(svcName string, logger *log.Logger) []*common.ServiceInstance{

	//该服务已监控并缓存
	instanceList, ok := consulClt.instancesMap.Load(svcName)
	if ok {
		return instanceList.([]*common.ServiceInstance)
	}

	//申请锁
	consulClt.mutex.Lock()
	defer consulClt.mutex.Unlock()
	// 再次检查是否监控
	instanceList, ok = consulClt.instancesMap.Load(svcName)
	if ok {
		return instanceList.([]*common.ServiceInstance)
	}else{
		//注册监控
		go func() {
			params := make(map[string]interface{})
			params["type"] = "service"
			params["service"] = svcName
			plan, _ := watch.Parse(params)
			plan.Handler = func(u uint64, i interface{}) {
				if i == nil{
					return
				}

				v, ok := i.([]*api.ServiceEntry)
				if !ok {
					return //数据异常，忽略
				}

				if len(v) == 0{
					consulClt.instancesMap.Store(svcName, []*common.ServiceInstance{})
				}

				var healthServices []*common.ServiceInstance
				for _, service := range v {
					if service.Checks.AggregatedStatus() == api.HealthPassing {
						healthServices = append(healthServices, newServiceInstance(service.Service))
					}
				}
				consulClt.instancesMap.Store(svcName, healthServices)
			}
			defer plan.Stop()
			plan.Run(consulClt.config.Address)
		}()
	}


	// 根据服务名请求服务实例列表
	entries, _, err := consulClt.client.Service(svcName, "", false, nil)
	if err != nil{
		consulClt.instancesMap.Store(svcName, []*common.ServiceInstance{})
		if logger != nil{
			logger.Println("Discover Service Error!")
		}
		return nil
	}

	instances := make([]*common.ServiceInstance, len(entries))
	for i := 0; i< len(instances); i++{
		instances[i] = newServiceInstance(entries[i].Service)
	}
	consulClt.instancesMap.Store(svcName, instances)
	return instances
}

func newServiceInstance(service *api.AgentService) *common.ServiceInstance{
	rpcPort := service.Port - 1
	if service.Meta != nil{
		if rpcPortStr, ok := service.Meta["rpcPort"]; ok{
			rpcPort, _ = strconv.Atoi(rpcPortStr)
		}
	}

	return &common.ServiceInstance{
		Host: service.Address,
		Port: service.Port,
		GrpcPort: rpcPort,
		Weight: service.Weights.Passing,
	}
}