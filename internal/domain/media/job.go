package media

import (
	"context"
	"errors"
	"time"
)

var (
	ErrJobNotFound   = errors.New("job not found")
	ErrInvalidUserID = errors.New("invalid user id")
)

const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
)

type Job struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"user_id" db:"user_id"`
	Type         string    `json:"type" db:"type"`
	Status       string    `json:"status" db:"status"`
	Progress     int       `json:"progress" db:"progress"`
	ErrorMessage string    `json:"error_message" db:"error_message"`
	ResultKey    string    `json:"result_key" db:"result_key"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

//go:generate mockgen -source=job.go -destination=mocks/mock_job_repository.go -package=mocks
type JobRepository interface {
	Create(ctx context.Context, j Job) error
	GetByID(ctx context.Context, id string) (*Job, error)
	UpdateStatus(ctx context.Context, id string, status string, progress int, resultKey string, errMsg string) error
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]Job, error)
}
