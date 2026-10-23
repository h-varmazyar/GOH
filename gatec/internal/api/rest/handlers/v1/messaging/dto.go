package messaging

// sendMessageReq is the request body for sending a message.
type sendMessageReq struct {
	SenderID    string `json:"sender_id" binding:"required"`
	ClientID    string `json:"client_id" binding:"required"`
	ChatID      int64  `json:"chat_id" binding:"required,gt=0"`
	Content     string `json:"content" binding:"required"`
	ContentType string `json:"content_type"` // text, photo, document
}

// scheduleMessageReq is the request body for scheduling a message.
type scheduleMessageReq struct {
	SenderID        string `json:"sender_id" binding:"required"`
	ClientID        string `json:"client_id" binding:"required"`
	ChatID          int64  `json:"chat_id" binding:"required,gt=0"`
	Content         string `json:"content" binding:"required"`
	ContentType     string `json:"content_type"`
	ScheduledAtUnix int64  `json:"scheduled_at_unix" binding:"required,gt=0"`
}

// grantAccessReq is the request body for granting access.
type grantAccessReq struct {
	SenderID      string `json:"sender_id" binding:"required"`
	ClientID      string `json:"client_id" binding:"required"`
	Permission    string `json:"permission" binding:"required"` // send, schedule, send_to_all_chats
	GrantedBy     string `json:"granted_by" binding:"required"`
	Reason        string `json:"reason"`
	ExpiresAtUnix int64  `json:"expires_at_unix"` // 0 = no expiration
}

// errorResp is a generic error response.
type errorResp struct {
	Error string `json:"error"`
}
