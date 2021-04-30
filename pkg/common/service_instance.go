package common

type ServiceInstance struct {
	ID string
	Host string
	Port int
	Weight int			//权重
	CurWeight int		//当前权重
	GrpcPort int
}
