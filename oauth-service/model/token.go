package model

import "time"

type OAuth2Token struct {
	TokenType string	//令牌类型
	TokenValue string	//令牌值
	ExpiresTime *time.Time	//令牌过期时间
	RefreshToken *OAuth2Token	//刷新令牌
}

//只有设置了过期时间并且过期了才为true
func (oauth2Token *OAuth2Token) IsExpired() bool{
	return oauth2Token.ExpiresTime != nil && oauth2Token.ExpiresTime.Before(time.Now())
}

//关联用户和客户端
type OAuth2Details struct {
	Client *ClientDetails
	User *UserDetails
}