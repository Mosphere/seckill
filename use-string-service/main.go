package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-kit/kit/circuitbreaker"
	uuid "github.com/satori/go.uuid"
	"net/http"
	"os"
	"os/signal"
	"seckill/pkg/discover"
	"seckill/pkg/loadbalance"
	"seckill/use-string-service/config"
	"seckill/use-string-service/endpoint"
	"seckill/use-string-service/service"
	"seckill/use-string-service/transport"
	"syscall"
)

func main() {
	//获取命令行参数
	var (
		consulHost  = flag.String("consul.host", "127.0.0.1", "consul host")
		consulPort  = flag.String("consul.port", "8500", "consul port")
		serviceName = flag.String("service.name", "use string", "service name")
		serviceHost = flag.String("service.host", "127.0.0.1", "service host")
		servicePort = flag.String("service.port", "8057", "service host")
	)
	flag.Parse()
	var svc service.Service

	var client *discover.DiscoveryClient
	client = discover.New(*consulHost,*consulPort)
	svc = service.NewService(client, &loadbalance.RandomLoadBalance{})
	useStringEndpoint := endpoint.MakeUseStringService(svc)
	//添加hystrix服务熔断中间件
	useStringEndpoint = circuitbreaker.Hystrix(service.StringSvcCmdName)(useStringEndpoint)
	healthCheckEndpoint := endpoint.MakeHealthCheckEndpoint(svc)
	endpoints := endpoint.UseStringEndpoints{
		UseStringEndpoint:   useStringEndpoint,
		HealthCheckEndpoint: healthCheckEndpoint,
	}
	ctx := context.Background()

	r := transport.MakeHttpHandler(ctx ,endpoints, config.KitLogger)
	UUID, _ := uuid.NewV4()
	instanceId := *serviceName + "-" + UUID.String()
	var errC = make(chan error)
	//http server
	go func() {
		config.Logger.Println("Http server start at port: ", *servicePort)
		bl := client.Register(instanceId, *serviceHost, *servicePort,"/health", *serviceName,2, nil, nil, config.Logger)
		if !bl{
			config.Logger.Printf("use-string-service for service %s failed.", serviceName)
			// 注册失败，服务启动失败
			os.Exit(-1)
		}
		errC<- http.ListenAndServe(":"+ *servicePort, r)
	}()

	go func() {
		// 监控系统关闭信号
		c := make(chan os.Signal,1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errC<- fmt.Errorf("%s", <-c)
	}()

	err := <-errC
	//服务退出取消注册
	client.DeRegister(instanceId, config.Logger)
	config.Logger.Println(err)
}
