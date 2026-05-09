package model

import "github.com/google/uuid"

type Quota struct {
	ID         int       `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	UsedBytes  int       `json:"used_bytes" db:"used_bytes"`
	TotalBytes int       `json:"total_bytes" db:"total_bytes"`
	GifCount   int       `json:"gif_count" db:"gif_count"`
	GitCount   int       `json:"gif_limit" db:"gif_limit"`
}
