package auth

type LoginByPasswordReq struct {
	Mobile   string `json:"mobile" validate:"required,len=11,ir_phone"`
	Password string `json:"password" validate:"required"`
}

type RegisterReq struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Username  string `json:"username" validate:"required"`
	Mobile    string `json:"mobile" validate:"required,len=11,ir_phone"`
	Password  string `json:"password" validate:"required,min=8"`
}
