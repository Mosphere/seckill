package service

// Service Define a service interface
type Service interface {
	// HealthCheck check service health status
	Health() bool
}

type OAuthService struct {

}

func NewOAuthService() *OAuthService {
	return &OAuthService{}
}
//健康检查
func (svc *OAuthService) Health() bool{
	return true
}

// ServiceMiddleware define service middleware
type ServiceMiddleware func(Service) Service
