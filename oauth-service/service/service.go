package service

// Service Define a service interface
type Service interface {
	// HealthCheck check service health status
	Health() bool
	SimpleData(string) string
}

type OAuthService struct {
}

func NewOAuthService() *OAuthService {
	return &OAuthService{}
}

func (s *OAuthService) SimpleData(username string) string {
	return "hello " + username + " ,simple data, with simple authority"
}

//健康检查
func (svc *OAuthService) Health() bool {
	return true
}

// ServiceMiddleware define service middleware
type ServiceMiddleware func(Service) Service
