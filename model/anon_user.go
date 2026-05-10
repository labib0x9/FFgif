package model

import (
	"time"

	"github.com/google/uuid"
)

type AnonUser struct {
	Id        uuid.UUID  `json:"id" db:"id"`
	Username  string     `json:"username"      db:"username"`
	Fullname  string     `json:"fullname"      db:"fullname"`
	Role      string     `json:"role"          db:"role"`
	CreatedAt time.Time  `json:"created_at"    db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"    db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"    db:"deleted_at"`
}
