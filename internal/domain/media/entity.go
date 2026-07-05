package media

import (
	"time"

	"github.com/google/uuid"
	minio_go "github.com/minio/minio-go/v7"
)

var (
	ContentLength = 512
)

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

type LastUpload struct {
	ID           int        `db:"id"            json:"id"`
	UserID       uuid.UUID  `db:"user_id"       json:"user_id"`
	FileKey      string     `db:"file_key"      json:"key"`
	Filename     string     `db:"file_name" json:"filename"`
	ContentType  string     `db:"content_type"  json:"content_type"`
	SizeBytes    *int64     `db:"size_bytes"    json:"size_bytes,omitempty"`
	DurationSec  *float64   `db:"duration_sec"  json:"duration_sec,omitempty"`
	ThumbnailURL *string    `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
	UploadedAt   time.Time  `db:"uploaded_at"   json:"uploaded_at"`
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

type Info struct {
	Size        int64
	ContentType string
	UploadedAt  time.Time
}

type Object struct {
	*minio_go.Object
}

type UploadResult struct {
	Url      string `json:"upload_url"`
	Key      string `json:"key"`
	ExpireIn int64  `json:"expires_in"`
}

type StreamResult struct {
	PresignedUrl string `json:"url"`
	ExpireIn     int    `json:"expires_in"`
}
