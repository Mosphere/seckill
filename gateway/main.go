package main

import (
	"flag"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/hashicorp/consul/api"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func NewReverseProxy(client *api.Client, logger log.Logger) *httputil.ReverseProxy{
	direct := func(req *http.Request) {
		reqPath := req.URL.Path
		pathArr := strings.Split(reqPath, "/")
		//服务名
		svcName := pathArr[1]
		logger.Log("svcName:", svcName)
		//调用consul的api查询svcName对应的服务列表
		svcList, _, err := client.Catalog().Service(svcName,"", nil)
		if err != nil{
			logger.Log("ReverseProxy failed", "query service instance error:", err.Error())
			return
		}

		if len(svcList) == 0{
			logger.Log("ReverseProxy failed", "no such service instance:", svcName)
			return
		}
		//去掉svcName后重新组织请求路径
		destPath := strings.Join(pathArr[2:], "/")
		target := svcList[rand.Intn(len(svcList))]
		logger.Log("service id", target.ServiceName)

		//设置代理服务信息
		req.URL.Scheme = "http"
		req.URL.Host = fmt.Sprintf("%s:%d", target.ServiceAddress, target.ServicePort)
		req.URL.Path = "/" + destPath
	}
	 return &httputil.ReverseProxy{Director: direct}
}

func main(){
	//创建环境变量
	var (
		consulHost = flag.String("consul.host", "127.0.0.1", "consul server ip address")
		consulPort = flag.String("consul.port", "8500", "consul server port")
	)
	flag.Parse()

	//创建日志组件
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	consulConfig := api.DefaultConfig()
	consulConfig.Address = "http://" + *consulHost + ":" + *consulPort
	consulClient, err := api.NewClient(consulConfig)
	if err != nil{
		logger.Log("err :", err)
		os.Exit(1)
	}
	proxy := NewReverseProxy(consulClient, logger)
	errC := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errC <- fmt.Errorf("%s", <-c)
	}()

	//监听http端口
	go func() {
		logger.Log("transport", "HTTP", "addr", "9009")
		errC<- http.ListenAndServe(":9009", proxy)
	}()

	logger.Log("exit", <-errC)
}
