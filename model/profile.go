package model

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	Id         int64     `json:"id" db:"id"`
	UserId     uuid.UUID `json:"user_id" db:"user_id"`
	ProfilePic string    `json:"profile_pic"   db:"profile_pic"`
	CreatedAt  time.Time `json:"created_at"    db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"    db:"updated_at"`
}

type ProfileResp struct {
	ProfilePic string `json:"avatar_url"   db:"profile_pic"`
	Username   string `json:"username"      db:"username"`
	Fullname   string `json:"fullname"      db:"fullname"`
	Email      string `json:"email"         db:"email"`
	IsVerified bool   `json:"verified"   db:"is_verified"`
}
