package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"seckill/pkg/discover"
	"seckill/string-service/config"
	"seckill/string-service/endpoint"
	"seckill/string-service/service"
	"seckill/string-service/transport"
	"syscall"

	uuid "github.com/satori/go.uuid"
)

func main() {

	// 获取命令行参数
	var (
		servicePort = flag.String("service.port", "10085", "service port")
		serviceHost = flag.String("service.host", "127.0.0.1", "service host")
		consulPort  = flag.String("consul.port", "8500", "consul port")
		consulHost  = flag.String("consul.host", "127.0.0.1", "consul host")
		serviceName = flag.String("service.name", "string", "service name")
	)

	flag.Parse()

	ctx := context.Background()
	errChan := make(chan error)
	var discoveryClient = discover.New(*consulHost, *consulPort)

	/*if err != nil{
		config.Logger.Println("Get Consul Client failed")
		os.Exit(-1)

	}*/
	var svc = service.StringService{}
	stringEndpoint := endpoint.MakeStringEndpoint(svc)

	//创建健康检查的Endpoint
	healthEndpoint := endpoint.MakeHealthCheckEndpoint(svc)

	//把算术运算Endpoint和健康检查Endpoint封装至StringEndpoints
	endpts := endpoint.StringEndpoints{
		StringEndpoint:      stringEndpoint,
		HealthCheckEndpoint: healthEndpoint,
	}

	//创建http.Handler
	r := transport.MakeHttpHandler(ctx, endpts, config.KitLogger)
	UUID := uuid.NewV4()
	instanceId := *serviceName + "-" + UUID.String()

	//http server
	go func() {
		config.Logger.Println("Http Server start at port:" + *servicePort)
		//启动前执行注册
		if !discoveryClient.Register(instanceId, *serviceHost, *servicePort, "/health", *serviceName, 3, nil, nil, config.Logger) {
			config.Logger.Printf("string-service for service %s failed.", *serviceName)
			// 注册失败，服务启动失败
			os.Exit(-1)
		}
		handler := r
		errChan <- http.ListenAndServe(":"+*servicePort, handler)
	}()

	go func() {
		//监控系统关闭信号
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	err := <-errChan
	//服务退出取消注册
	discoveryClient.DeRegister(instanceId, config.Logger)
	config.Logger.Println(err)
}
