package chats

import "time"

type createGroupRequest struct {
	Title string `json:"title" example:"Команда проекта" minLength:"1" maxLength:"50"`
} //@name CreateGroupRequest

type createChatResponse struct {
	ID int `json:"id" example:"13"`
} //@name CreateChatResponse

type createDirectRequest struct {
	UserID int `json:"user_id" example:"39"`
} //@name CreateDirectRequest

type muteRequest struct {
	Muted bool `json:"muted" example:"false"`
} //@name MuteRequest

type chatResponse struct {
	ID          int       `json:"id" example:"13"`
	Title       *string   `json:"title" example:"Команда проекта" extensions:"x-nullable"`
	Type        string    `json:"type" example:"group" enums:"direct,group"`
	CreatedAt   time.Time `json:"created_at" example:"2026-10-22T12:00:00Z"`
	UnreadCount int       `json:"unread_count" example:"3"`
	PeerLogin   *string   `json:"peer_login,omitempty" example:"sonya"`
	Muted       bool      `json:"muted" example:"false"`
} //@name Chat

type addMemberRequest struct {
	UserID int `json:"user_id" example:"40"`
} //@name AddMemberRequest

type memberResponse struct {
	UserID int    `json:"user_id" example:"39"`
	Login  string `json:"login" example:"sonya"`
	Role   string `json:"role" example:"member" enums:"admin,member"`
} //@name Member
