package auth

type LoginByPasswordReq struct {
	Mobile   string
	Password string
}

type LoginByPasswordResp struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	HashedID     string
}

type RegisterReq struct {
	FirstName string
	LastName  string
	Username  string
	Mobile    string
	Password  string
}

type RegisterResp struct {
	HashedID string
}
