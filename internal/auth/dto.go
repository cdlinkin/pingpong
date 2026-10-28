package auth

import "time"

type registerRequest struct {
	Login    string `json:"login" example:"kuroame" minLength:"3" maxLength:"25"`
	Email    string `json:"email" example:"kuroame@example.com" maxLength:"250"`
	Password string `json:"password" example:"supersecret" minLength:"8" maxLength:"72"`
} //@name RegisterRequest

type loginRequest struct {
	Email    string `json:"email" example:"kuroame@example.com"`
	Password string `json:"password" example:"supersecret"`
} //@name LoginRequest

type registerResponse struct {
	ID int `json:"id" example:"38"`
} //@name RegisterResponse

type loginResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
} //@name LoginResponse

type meResponse struct {
	ID        int       `json:"id" example:"38"`
	Login     string    `json:"login" example:"kuroame"`
	Email     string    `json:"email" example:"kuroame@example.com"`
	CreatedAt time.Time `json:"created_at" example:"2026-10-20T18:30:00Z"`
} //@name User
