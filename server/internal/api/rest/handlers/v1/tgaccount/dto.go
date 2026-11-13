package tgaccount

// --- Request DTOs ---

type createAccountReq struct {
	SessionType string `json:"session_type" binding:"required" example:"client"`
}

type submitPhoneReq struct {
	SessionID   string `json:"session_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	PhoneNumber string `json:"phone_number" binding:"required" example:"+989123456789"`
}

type submitBotTokenReq struct {
	SessionID string `json:"session_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	BotToken  string `json:"bot_token" binding:"required" example:"123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw"`
}

type submitCodeReq struct {
	SessionID string `json:"session_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code      string `json:"code" binding:"required" example:"12345"`
}

type submitPasswordReq struct {
	SessionID string `json:"session_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Password  string `json:"password" binding:"required" example:"my2fapassword"`
}

type createFilterReq struct {
	AccountID string `json:"account_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChatID    int64  `json:"chat_id" binding:"required" example:"-1003944858901"`
}

type reAuthReq struct {
	AccountID string `json:"account_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// --- Response DTOs ---

type createAccountResp struct {
	SessionID string `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	State     string `json:"next_step" example:"wait_phone"`
}

type stateResp struct {
	SessionID string `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	State     string `json:"next_step" example:"wait_code"`
}

type errorResp struct {
	Error string `json:"error" example:"session not found"`
}
