package chats

type createGroupRequest struct {
	Title string `json:"title"`
}

type createChatResponse struct {
	ID int `json:"id"`
}

type createDirectRequest struct {
	UserID int `json:"user_id"`
}
