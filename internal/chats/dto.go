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

type chatResponse struct {
	ID        int       `json:"id" example:"13"`
	Title     *string   `json:"title" example:"Команда проекта" extensions:"x-nullable"`
	Type      string    `json:"type" example:"group" enums:"direct,group"`
	CreatedAt time.Time `json:"created_at" example:"2026-10-22T12:00:00Z"`
} //@name Chat

type addMemberRequest struct {
	UserID int `json:"user_id" example:"40"`
} //@name AddMemberRequest
