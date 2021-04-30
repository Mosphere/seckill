package discover

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"seckill/pkg/bootstrap"
	"seckill/pkg/common"
	"seckill/pkg/loadbalance"
	"github.com/satori/go.uuid"
)

var ConsulClient IDiscoveryClient
var LoadBalance loadbalance.LoadBalance
var Logger *log.Logger
var NoInstanceExistedErr  = errors.New("no available client")
func init(){
	//实例化一个 Consul 客户端，此处实例化了原生态实现版本
	ConsulClient = New(bootstrap.DiscoverConfig.Host, bootstrap.DiscoverConfig.Port)
	if ConsulClient == nil{

	}
	LoadBalance = new(loadbalance.RandomLoadBalance)
	Logger = log.New(os.Stderr, "", log.LstdFlags)
}

//健康检查
func CheckHealth(w http.ResponseWriter, r *http.Request){
	Logger.Println("Health Check !")
	_, err := fmt.Fprintln(w, "Server is OK!")
	if err != nil {
		Logger.Println(err)
	}
}

/**
发现服务
 */
func DiscoveryService(svcName string) (*common.ServiceInstance, error){
	instances := ConsulClient.DiscoveryServices(svcName, Logger)
	 if len(instances) < 1 {
	 	Logger.Printf("no available client for %s.", svcName)
	 	return nil, NoInstanceExistedErr
	 }
	 return LoadBalance.SelectService(instances)
}

func Register() {
	//// 实例失败，停止服务
	if ConsulClient == nil {
		panic(0)
	}

	//判空 instanceId,通过 go.uuid 获取一个服务实例ID
	instanceId := bootstrap.DiscoverConfig.InstanceId

	if instanceId == "" {
		UUID, _ := uuid.NewV4()
		instanceId = bootstrap.DiscoverConfig.ServiceName + UUID.String()
	}

	if !ConsulClient.Register(instanceId, bootstrap.HttpConfig.Host, "/health",
		bootstrap.HttpConfig.Port, bootstrap.DiscoverConfig.ServiceName,
		bootstrap.DiscoverConfig.Weight,
		map[string]string{
			"rpcPort": bootstrap.RpcConfig.Port,
		}, nil, Logger) {
		Logger.Printf("register service %s failed.", bootstrap.DiscoverConfig.ServiceName)
		// 注册失败，服务启动失败
		panic(0)
	}
	Logger.Printf(bootstrap.DiscoverConfig.ServiceName+"-service for service %s success.", bootstrap.DiscoverConfig.ServiceName)

}

func DeRegister(){
	if ConsulClient == nil{
		panic(0)
	}

	instanceId := bootstrap.DiscoverConfig.InstanceId
	if instanceId == ""{
		UUID, _ := uuid.NewV4()
		instanceId = bootstrap.DiscoverConfig.ServiceName + "-" + UUID.String()
	}
	if !ConsulClient.DeRegister(instanceId, Logger){
		Logger.Printf("deregister for service %s failed.", bootstrap.DiscoverConfig.ServiceName)
		panic(0)
	}
}