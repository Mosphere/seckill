package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"seckill/oauth-service/config"
	"seckill/oauth-service/endpoints"
	"seckill/oauth-service/model"
	"seckill/oauth-service/service"
	"seckill/oauth-service/transport"
	"seckill/pkg/discover"
	"syscall"

	"github.com/astaxie/beego/logs"
	uuid "github.com/satori/go.uuid"
)

func main() {
	var (
		servicePort = flag.String("service.port", "10098", "service port")
		serviceHost = flag.String("service.host", "127.0.0.1", "service host")
		serviceName = flag.String("service.name", "oauth", "service name")

		consulHost = flag.String("consul.host", "127.0.0.1", "consul host")
		consulPort = flag.String("consul.port", "8500", "consul port")
	)
	flag.Parse()
	var tokenGranter service.TokenGranter
	var tokenSvc service.TokenService
	var userDetailsSvc service.UserDetailsService
	var clientDetailSvc service.ClientDetailsService
	userDetailsSvc = service.NewInMemoryUserDetailsService([]*model.UserDetails{
		{
			Username:    "admin",
			Password:    "123456",
			UserId:      1,
			Authorities: []string{"admin"},
		},
		{
			Username:    "test",
			Password:    "123456",
			UserId:      2,
			Authorities: []string{"test"},
		},
	})

	clientDetailSvc = service.NewInMemoryClientDetailsService([]*model.ClientDetails{
		{
			ClientId:                   "clientId",
			ClientSecret:               "clientSecret",
			AccessTokenValid:           1800,
			RefreshAccessTokenValidity: 7200,
			RegisteredRedirectUri:      "http://127.0.0.1",
			AuthorizedGrantTypes:       []string{"password", "refresh_token"},
		},
	})
	tokenEnhancer := service.NewJwtTokenEnhancer("secret")
	tokenStore := service.NewJwtTokenStore(tokenEnhancer.(*service.JwtTokenEnhancer))
	tokenSvc = service.NewTokenService(tokenStore, tokenEnhancer)
	tokenGranter = service.NewComposeTokenGranter(map[string]service.TokenGranter{
		"password": service.NewPasswordTokenGranter("password", userDetailsSvc, tokenSvc),
	})

	var discoveryClient = discover.New(*consulHost, *consulPort)
	ctx := context.Background()
	var svc = service.NewOAuthService()
	endpts := endpoints.OauthEndpoints{
		TokenEndpoint:      endpoints.MakeTokenEndpoint(tokenGranter),
		CheckTokenEndpoint: endpoints.MakeCheckTokenEndpoint(tokenSvc),
		HealthEndpoint:     endpoints.MakeHealthEndpoint(svc),
	}
	//创建httpHandler
	r := transport.MakeHttpHandler(ctx, endpts, clientDetailSvc, config.KitLogger)
	instanceId := *serviceName + "-" + uuid.NewV4().String()
	logs.Info("instanceId: ", instanceId)
	errC := make(chan error)
	go func() {
		logs.Info("servicePort: ", *servicePort)
		//config.Logger.Println("service listen port at: ")
		//注册服务
		if !discoveryClient.Register(instanceId, *serviceHost, *servicePort, "/health", *serviceName, 1, nil, nil, &config.Logger) {
			//注册失败
			os.Exit(-1) //可以被signal捕获
		}
		errC <- http.ListenAndServe(":"+*servicePort, r)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		logs.Info(2)
		errC <- fmt.Errorf("%s", <-c)
	}()

	err := <-errC
	//服务退出取消注册
	discoveryClient.DeRegister(instanceId, &config.Logger)
	logs.Info(1)
	config.Logger.Println(err)
}
