package chats

type createGroupRequest struct {
	Title string `json:"title"`
}

type createGroupResponse struct {
	ID int `json:"id"`
}
