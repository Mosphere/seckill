package loadbalance

import (
	"errors"
	"math/rand"
	"seckill/pkg/common"
)

//负载均衡器
type LoadBalance interface {
	SelectService([]*common.ServiceInstance) (*common.ServiceInstance, error)
}

type RandomLoadBalance struct {}


func (loadBalance *RandomLoadBalance)SelectService(services []*common.ServiceInstance) (*common.ServiceInstance, error){
	if services == nil || len(services) == 0{
		return nil, errors.New("service instances are not exist")
	}
	return services[rand.Intn(len(services))], nil
}

type WeightRoundLoadBalance struct{}

func (loadBalance *WeightRoundLoadBalance)SelectService(services []*common.ServiceInstance) (best *common.ServiceInstance, err error){
	if services == nil || len(services) == 0{
		return nil, errors.New("service instances are not exist")
	}
	total := 0
	for i:= 0; i< len(services);i++{
		w := services[i]
		w.CurWeight += w.Weight
		total += w.Weight
		if best == nil || w.CurWeight > best.CurWeight{
			best = w
		}
	}
	if best == nil{
		return nil,nil
	}
	best.CurWeight -= total
	return best, nil
}

