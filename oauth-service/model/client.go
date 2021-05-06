package model

type ClientDetails struct {
	ClientId string
	ClientSecret string
	AccessTokenValid int64
	RefreshAccessTokenValidity int64
	AuthorizedGrantTypes []string
	RegisteredRedirectUri string
}
