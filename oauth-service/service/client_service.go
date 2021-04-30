package service

import (
	"context"
	"errors"
	"seckill/oauth-service/model"
)

var (
	ErrClientNotExist = errors.New("clientId is not exist")
	ErrClientSecret = errors.New("invalid clientSecret")
)
type ClientDetailsService interface {
	GetClientDetailByClientId(ctx context.Context, clientId string, clientSecret string) (*model.ClientDetails,error)
}

type InMemoryClientDetailsService struct{
	ClientDetailsDict map[string]*model.ClientDetails
}

func NewInMemoryClientDetailsService(clientDetailsList []*model.ClientDetails) ClientDetailsService{
	clientDetailsDict := make(map[string]*model.ClientDetails)
	if len(clientDetailsList) > 0{
		for _, v := range clientDetailsList{
			clientDetailsDict[v.ClientId] = v
		}
	}
	return &InMemoryClientDetailsService{
		ClientDetailsDict: clientDetailsDict,
	}
}

func (clientDetailSvc *InMemoryClientDetailsService) GetClientDetailByClientId(ctx context.Context, clientId string, clientSecret string) (*model.ClientDetails,error){
	//根据clientId获取clientDetails
	clientDetails, ok := clientDetailSvc.ClientDetailsDict[clientId]
	if !ok{
		return nil, ErrClientNotExist
	}

	//检测密码是否正确
	if clientDetails.ClientSecret != clientSecret{
		return nil, ErrClientSecret
	}
	return clientDetails, nil
}