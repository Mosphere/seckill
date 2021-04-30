package service

import (
	"context"
	"errors"
	"github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"
	"net/http"
	"seckill/oauth-service/model"
	"time"
)

var (
	ErrNotSupportGrantType = errors.New("grant type is not supported")
	ErrNotSupportOperation = errors.New("no support operation")
	ErrInvalidUsernameAndPasswordRequest = errors.New("invalid username, password")
	ErrExpiredToken =  errors.New("token expired")
)
type TokenGranter interface {
	Grant(ctx context.Context, grantType string, client *model.ClientDetails, reader *http.Request)(*model.OAuth2Token, error)
}

//
type PasswordTokenGranter struct{
	grantType string
	UserDetailsSvc UserDetailsService
	TokenService TokenService
}

func NewPasswordTokenGranter(grantType string, userDetailsSvc UserDetailsService, tokenSvc TokenService) *PasswordTokenGranter{
	return &PasswordTokenGranter{
		grantType: grantType,
		UserDetailsSvc: userDetailsSvc,
		TokenService: tokenSvc,
	}
}
func (tokenGranter *PasswordTokenGranter) Grant(ctx context.Context, grantType string, client *model.ClientDetails, reader *http.Request)(*model.OAuth2Token, error){
	if grantType != tokenGranter.grantType{
		return nil, ErrNotSupportGrantType
	}

	userName := reader.FormValue("username")
	password := reader.FormValue("password")
	userDetails, err := tokenGranter.UserDetailsSvc.GetUserDetailByUsername(ctx, userName, password)
	if err != nil{
		return nil, err
	}

	oAuth2Details := &model.OAuth2Details{
		Client: client,
		User: userDetails,
	}
	return tokenGranter.TokenService.CreateAccessToken(oAuth2Details)
}

type ComposeTokenGranter struct {
	TokenGranterDict map[string]TokenGranter
}

func NewComposeTokenGranter(tokenGranterDict map[string]TokenGranter) TokenGranter{
	return &ComposeTokenGranter{
		TokenGranterDict: tokenGranterDict,
	}
}

func (composeTokenGranter *ComposeTokenGranter) Grant(ctx context.Context, grantType string, client *model.ClientDetails, reader *http.Request)(*model.OAuth2Token, error){
	//检测客户端是否允许这种操作
	var isSupport bool
	for _, v := range client.AuthorizedGrantTypes{
		if v == grantType{
			isSupport = true
		}
	}
	if !isSupport{
		return nil, ErrNotSupportOperation
	}

	//查找具体的授权类型实现方法
	dispatchTokenGranter, ok := composeTokenGranter.TokenGranterDict[grantType]
	if !ok{
		return nil, ErrNotSupportGrantType
	}
	return dispatchTokenGranter.Grant(ctx, grantType, client, reader)
}
/*type OAuth2Details struct {
	Client *model.ClientDetails
	User model.UserDetails
}*/
type TokenService interface {
	//根据用户信息和客户端信息获取访问令牌
	GetAccessToken(details *model.OAuth2Details) (*model.OAuth2Token, error)
	ReadAccessToken(tokenValue string) (*model.OAuth2Token, error)

	//根据token值获取用户信息和客户端信息
	GetOAuth2DetailsByAccessToken(tokenValue string) (*model.OAuth2Details, error)
	// RefreshAccessToken(refreshTokenValue string) (*model.OAuth2Details, error)
	CreateAccessToken(details *model.OAuth2Details) (*model.OAuth2Token, error)
}

type TokenStore interface {
	//根据客户端信息获取令牌结构体
	GetAccessToken(*model.OAuth2Details) (*model.OAuth2Token, error)
	//更具令牌值获取令牌结构体
	ReadAccessToken(string) (*model.OAuth2Token, error)
	// 根据令牌值获取令牌对应的客户端和用户信息
	ReadOAuth2Details(tokenValue string) (*model.OAuth2Details, error)
	StoreAccessToken(token *model.OAuth2Token, details *model.OAuth2Details)
	RemoveAccessToken(tokenValue string) error
	RemoveRefreshToken(tokenValue string) error
}
type DefaultTokenService struct {
	tokenStore TokenStore
	tokenEnhancer TokenEnhancer
}

func NewTokenService(tokenStore TokenStore, tokenEnhancer TokenEnhancer) TokenService{
	return &DefaultTokenService{
		tokenStore: tokenStore,
		tokenEnhancer: tokenEnhancer,
	}
}

func (tokenService *DefaultTokenService)GetOAuth2DetailsByAccessToken(tokenValue string) (*model.OAuth2Details, error){
	accessToken, err := tokenService.ReadAccessToken(tokenValue)
	if err == nil{
		if accessToken.IsExpired(){
			return nil, ErrExpiredToken
		}
		return tokenService.tokenStore.ReadOAuth2Details(tokenValue)
	}
	return nil, err
}

func (tokenService *DefaultTokenService) CreateAccessToken(details *model.OAuth2Details) (*model.OAuth2Token, error){
	existToken, err := tokenService.tokenStore.GetAccessToken(details)
	if err != nil{
		return nil, err
	}

	var refreshToken *model.OAuth2Token
	if existToken != nil{
		if !existToken.IsExpired(){
			//如果token没过期，将其存起来
			tokenService.tokenStore.StoreAccessToken(existToken, details)
			return existToken, err
		}

		//token已过期要移除
		err = tokenService.tokenStore.RemoveAccessToken(existToken.TokenValue)
		if err != nil{
			return nil, err
		}

		if existToken.RefreshToken != nil{
			refreshToken = existToken.RefreshToken
			err = tokenService.tokenStore.RemoveRefreshToken(refreshToken.TokenValue)
			if err != nil{
				return nil, err
			}
		}

	}
	if refreshToken == nil || refreshToken.IsExpired(){
		//生成刷新令牌
		refreshToken, err = tokenService.createRefreshToken(details)
		if err != nil{
			return nil, err
		}
	}
	//生成新的访问令牌
	accessToken, err := tokenService.createAccessToken(refreshToken, details)
	if err == nil{
		//保存新的生成令牌
		tokenService.tokenStore.StoreAccessToken(accessToken, details)
		tokenService.tokenStore.StoreAccessToken(refreshToken, details)
	}
	return accessToken, err
}

func (tokenService *DefaultTokenService) createAccessToken(refreshToken *model.OAuth2Token, details *model.OAuth2Details) (*model.OAuth2Token, error){
	validity := details.Client.RefreshAccessTokenValidity
	expiredTime := time.Now().Add(time.Duration(validity))
	UUID, _ := uuid.NewV4()
	tokenValue := UUID.String()
	accessToken := &model.OAuth2Token{
		ExpiresTime: &expiredTime,
		TokenValue: tokenValue,
		RefreshToken: refreshToken,
	}
	if tokenService.tokenEnhancer != nil{
		return tokenService.tokenEnhancer.Enhance(accessToken, details)
	}
	return accessToken, nil
}
func (tokenService *DefaultTokenService) createRefreshToken(details *model.OAuth2Details) (*model.OAuth2Token, error){
	validity := details.Client.RefreshAccessTokenValidity
	expiredTime := time.Now().Add(time.Duration(validity))
	UUID, _ := uuid.NewV4()
	tokenValue := UUID.String()
	refreshToken := &model.OAuth2Token{
		TokenValue: tokenValue,
		ExpiresTime: &expiredTime,
	}
	if tokenService.tokenEnhancer != nil{
		return tokenService.tokenEnhancer.Enhance(refreshToken, details)
	}
	return refreshToken, nil
}

func (tokenService *DefaultTokenService)GetAccessToken(details *model.OAuth2Details)(*model.OAuth2Token, error){
	return tokenService.tokenStore.GetAccessToken(details)
}

func (tokenService *DefaultTokenService)ReadAccessToken(tokenValue string)(*model.OAuth2Token, error){
	return tokenService.tokenStore.ReadAccessToken(tokenValue)
}

type OAuth2TokenCustomClaims struct {
	UserDetails *model.UserDetails
	ClientDetails *model.ClientDetails
	RefreshToken *model.OAuth2Token
	jwt.StandardClaims
}

type TokenEnhancer interface {
	Enhance(oAuth2Token *model.OAuth2Token, details *model.OAuth2Details) (*model.OAuth2Token, error)
	Extract(tokenValue string) (*model.OAuth2Token, *model.OAuth2Details, error)
}

type JwtTokenEnhancer struct {
	secretKey []byte
}

func NewJwtTokenEnhancer(secret string) TokenEnhancer{
	return &JwtTokenEnhancer{
		secretKey: []byte(secret),
	}
}

type JwtTokenStore struct {
	jwtTokenEnhancer *JwtTokenEnhancer
}

func NewJwtTokenStore(enhancer *JwtTokenEnhancer) TokenStore{
	return &JwtTokenStore{
		jwtTokenEnhancer: enhancer,
	}
}

func (tokenStore *JwtTokenStore) StoreAccessToken(oauth2Token *model.OAuth2Token, oauth2Details *model.OAuth2Details){

}

func (tokenStore *JwtTokenStore) ReadAccessToken(tokenValue string) (*model.OAuth2Token, error) {
	oauth2Token, _, err := tokenStore.jwtTokenEnhancer.Extract(tokenValue)
	return oauth2Token, err

}

// 根据令牌值获取令牌对应的客户端和用户信息
func (tokenStore *JwtTokenStore) ReadOAuth2Details(tokenValue string) (*model.OAuth2Details, error) {
	_, oauth2Details, err := tokenStore.jwtTokenEnhancer.Extract(tokenValue)
	return oauth2Details, err

}

// 根据客户端信息和用户信息获取访问令牌
func (tokenStore *JwtTokenStore) GetAccessToken(oauth2Details *model.OAuth2Details) (*model.OAuth2Token, error) {
	return nil, ErrNotSupportOperation
}

// 移除存储的访问令牌
func (tokenStore *JwtTokenStore) RemoveAccessToken(tokenValue string) error{
	return nil
}

// 存储刷新令牌
func (tokenStore *JwtTokenStore) StoreRefreshToken(oauth2Token *model.OAuth2Token, oauth2Details *model.OAuth2Details) {

}

// 移除存储的刷新令牌
func (tokenStore *JwtTokenStore) RemoveRefreshToken(oauth2Token string) error{
	return nil
}

// 根据令牌值获取刷新令牌
func (tokenStore *JwtTokenStore) ReadRefreshToken(tokenValue string) (*model.OAuth2Token, error) {
	oauth2Token, _, err := tokenStore.jwtTokenEnhancer.Extract(tokenValue)
	return oauth2Token, err
}
func (enhancer *JwtTokenEnhancer)Enhance(oAuth2Token *model.OAuth2Token,details *model.OAuth2Details) (*model.OAuth2Token, error){
	userDetails := details.User
	clientDetails := details.Client
	expireTime := oAuth2Token.ExpiresTime
	//指控jwt中敏感字段
	clientDetails.ClientSecret = ""
	userDetails.Password = ""
	claims := &OAuth2TokenCustomClaims{
		UserDetails: userDetails,
		ClientDetails: clientDetails,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer: "System",
		},
	}
	if oAuth2Token.RefreshToken != nil{
		claims.RefreshToken = oAuth2Token.RefreshToken
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenValue, err := token.SignedString(enhancer.secretKey)
	if err == nil{
		oAuth2Token.TokenType = "jwt"
		oAuth2Token.TokenValue = tokenValue
		return oAuth2Token, nil
	}
	return nil, err
}

func (enhancer JwtTokenEnhancer) Extract(tokenValue string) (*model.OAuth2Token, *model.OAuth2Details, error){
	token, err := jwt.ParseWithClaims(tokenValue, &OAuth2TokenCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return enhancer.secretKey, nil
	})
	if err != nil{
		return nil, nil, err
	}
	claims := token.Claims.(*OAuth2TokenCustomClaims)
	expireTime := time.Unix(claims.ExpiresAt, 0)
	return  &model.OAuth2Token{
		RefreshToken: claims.RefreshToken,
		TokenValue: tokenValue,
		ExpiresTime: &expireTime,
	},
	&model.OAuth2Details{
		User: claims.UserDetails,
		Client: claims.ClientDetails,
	},nil
}



