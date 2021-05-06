package service

import (
	"context"
	"errors"
	"seckill/oauth-service/model"
)

var (
	ErrUserNotExist = errors.New("username is not exist")
	ErrPassword = errors.New("invalid password")
)
type UserDetailsService interface {
	GetUserDetailByUsername(ctx context.Context, username, password string) (*model.UserDetails, error)
}
type InMemoryUserDetailsService struct{
	UserDetailsDict map[string]*model.UserDetails
}

func NewInMemoryUserDetailsService(userDetailsList []*model.UserDetails) UserDetailsService{
	userDetailsSvcDict := make(map[string]*model.UserDetails)
	if len(userDetailsList) > 0{
		for _, v := range userDetailsList{
			userDetailsSvcDict[v.Username] = v
		}
	}
	return &InMemoryUserDetailsService{
		UserDetailsDict: userDetailsSvcDict,
	}
}

func (userDetailSvc *InMemoryUserDetailsService) GetUserDetailByUsername(ctx context.Context, username, password string) (*model.UserDetails, error){
	userDetail,ok := userDetailSvc.UserDetailsDict[username]
	if !ok{
		return nil, ErrUserNotExist
	}
	if userDetail.Password != password{
		return nil, ErrPassword
	}
	return userDetail, nil
}

