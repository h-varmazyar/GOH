package tabdeal

type UpdateAuthTokenReq struct {
	Token string `json:"token" binding:"required"`
}
