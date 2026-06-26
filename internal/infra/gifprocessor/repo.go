package gifprocessor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/pkg/ffmpeg"
	"github.com/labib0x9/ffgif/pkg/random"
)

type Fmeg struct {
	minio media.UploaderRepository
}

func NewFmeg(minioRepo media.UploaderRepository) *Fmeg {
	return &Fmeg{
		minio: minioRepo,
	}
}

func (f *Fmeg) Process(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (string, error) {
	inputPath := filepath.Join(os.TempDir(), Key+"_input."+f.getContentType(ctx, Key))
	outputPath := filepath.Join(os.TempDir(), Key+"_output.gif")
	palettePath := filepath.Join(os.TempDir(), Key+"_palette.png")

	defer os.Remove(inputPath)
	defer os.Remove(outputPath)
	defer os.Remove(palettePath)

	if err := f.minio.Download(ctx, Key, inputPath); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	if FPS == 0 {
		FPS = 10
	}

	if Width == 0 {
		Width = 480
	}

	loop := "-1"
	if Loop {
		loop = "0"
	}

	runner := ffmpeg.NewFFmpeg(ctx, inputPath, outputPath, palettePath, Width, FPS, Start, End, loop)
	if err := runner.Run(); err != nil {
		//
		return "", fmt.Errorf("ffmpeg failed")
	}

	gifKey := random.GenerateRandomID().String() + "_output.gif"
	if err := f.minio.Upload(ctx, gifKey, outputPath, "image/gif"); err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	return gifKey, nil
}

func (f *Fmeg) getContentType(ctx context.Context, key string) string {
	info, err := f.minio.StatObject(ctx, key)
	if err != nil {
		return ""
	}
	switch info.ContentType {
	case "video/mp4":
		return "mp4"
	case "video/quicktime":
		return "mov"
	case "video/x-matroska":
		return "mkv"
	case "video/webm":
		return "webm"
	case "video/avi":
		return "avi"
	}
	return ""
}
