package model

import (
	"time"
)

type Share struct {
	ID        string     `json:"id" db:"id"`
	GifKey    string     `json:"gif_key" db:"gif_key"`
	OwnerID   string     `json:"owner_id" db:"owner_id"`
	Token     string     `json:"token" db:"token"`
	Access    string     `json:"access" db:"access"`
	ExpiresAt *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

type SharedGifResp struct {
	GifResp
	ShareAccess string `json:"share_access" db:"access"`
}
