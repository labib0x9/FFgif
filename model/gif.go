package model

import "time"

type Gif struct {
	Key       string    `json:"key" db:"key"`
	UserId    string    `json:"user_id" db:"user_id"`
	Status    string    `json:"status" db:"status"`
	Persist   bool      `json:"persist" db:"persist"`
	Download  int       `json:"download" db:"download"`
	CreatedAt time.Time `json:"created_at"    db:"created_at"`
	UpdatedAt time.Time `json:"updated_at"    db:"updated_at"`
}

type GifResp struct {
	Key       string    `json:"key" db:"key"`
	Status    string    `json:"status" db:"status"`
	Persist   bool      `json:"persist" db:"persist"`
	Download  int       `json:"download" db:"download"`
	CreatedAt time.Time `json:"created_at"    db:"created_at"`
}