package model

import (
	"time"

	"github.com/google/uuid"
)

type LastUpload struct {
	ID           int        `db:"id"            json:"id"`
	UserID       uuid.UUID  `db:"user_id"       json:"user_id"`
	FileKey      string     `db:"file_key"      json:"key"`
	Filename     string     `db:"file_name" json:"filename"`
	ContentType  string     `db:"content_type"  json:"content_type"`
	SizeBytes    *int64     `db:"size_bytes"    json:"size_bytes,omitempty"`
	DurationSec  *float64   `db:"duration_sec"  json:"duration_sec,omitempty"`
	UploadedAt   time.Time  `db:"uploaded_at"   json:"uploaded_at"`
	ThumbnailURL *string    `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
	DeletedAt    *time.Time `db:"deleted_at"    json:"deleted_at,omitempty"`
	UpdatedAt    *time.Time `db:"updated_at"    json:"updated_at,omitempty"`
}

type LastUploadResp struct {
	UserID       uuid.UUID `db:"user_id"       json:"user_id"`
	FileKey      string    `db:"file_key"      json:"key"`
	Filename     string    `db:"file_name" json:"filename"`
	ContentType  string    `db:"content_type"  json:"content_type"`
	SizeBytes    *int64    `db:"size_bytes"    json:"size_bytes,omitempty"`
	DurationSec  *float64  `db:"duration_sec"  json:"duration_sec,omitempty"`
	UploadedAt   time.Time `db:"uploaded_at"   json:"uploaded_at"`
	ThumbnailURL *string   `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
}
