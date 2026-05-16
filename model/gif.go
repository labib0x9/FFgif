package model

import "time"

type Gif struct {
	Key          string    `json:"key" db:"key"`
	Name         string    `json:"name" db:"name"`
	UserId       string    `json:"user_id" db:"user_id"`
	Status       string    `json:"status" db:"status"`
	Persist      bool      `json:"persist" db:"persist"`
	Download     int       `json:"download" db:"download"`
	Url          string    `json:"url" db:"url"`
	ThumbnailUrl string    `json:"thumbnail_url" db:"thumbnail_url"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"    db:"updated_at"`
}

type GifResp struct {
	Key          string    `json:"key" db:"key"`
	Name         string    `json:"name" db:"name"`
	Status       string    `json:"status" db:"status"`
	Persist      bool      `json:"persist" db:"persist"`
	Url          string    `json:"url" db:"url"`
	ThumbnailUrl string    `json:"thumbnail_url" db:"thumbnail_url"`
	Download     int       `json:"download" db:"download"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
}

// name
// url
// thumbnail_url
// size_bytes
// width
// height
// duration
