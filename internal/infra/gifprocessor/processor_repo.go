package gifprocessor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/processor"
	"github.com/labib0x9/ffgif/pkg/ffmpeg"
	"github.com/labib0x9/ffgif/pkg/random"
)

type Fmeg struct {
	minio media.StorageRepository
}

func NewFmeg(minioRepo media.StorageRepository) *Fmeg {
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

	if err := f.minio.DownloadLocal(ctx, Key, inputPath); err != nil {
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

	runner := ffmpeg.NewGifConverter(ctx, inputPath, outputPath, palettePath, Width, FPS, Start, End, loop)
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

// download to local, validate the video file, convert to mp4 and returns the output path
// caller should delete the output video file
func (f *Fmeg) PreProcess(ctx context.Context, key string) (*processor.PrePrecessedResult, error) {
	inputPath := filepath.Join(os.TempDir(), "raw_video_"+key)
	outputPath := filepath.Join(os.TempDir(), "converted_"+key+".mp4")
	thumpOutputPath := filepath.Join(os.TempDir(), "thumb_"+key+".jpg")
	defer os.Remove(inputPath)
	defer os.Remove(outputPath)
	defer os.Remove(thumpOutputPath)

	if err := f.minio.DownloadLocalRawVideo(ctx, key, inputPath); err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	runner := ffmpeg.NewProbeExtractor(ctx, inputPath)
	result, err := runner.Run()
	if err != nil {
		return nil, err
	}

	if !result.IsValid() {
		return nil, fmt.Errorf("Invalid file")
	}

	mp4Converter := ffmpeg.NewMp4Converter(ctx, inputPath, outputPath, result.Stream[0].CodecName)
	if err := mp4Converter.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed to convert raw video to mp4")
	}

	videoThumbGenerator := ffmpeg.NewThumbGenerator(ctx, outputPath, thumpOutputPath)
	if err := videoThumbGenerator.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed to generate thumbnail from raw video")
	}

	videoKey := "converted_" + random.GenerateRandomID().String() + ".mp4"
	if err := f.minio.Upload(ctx, videoKey, outputPath, "video/mp4"); err != nil {
		return nil, fmt.Errorf("converted video upload failed: %w", err)
	}

	thumbKey := "thumpnail_" + random.GenerateRandomID().String() + ".jpg"
	if err := f.minio.Upload(ctx, thumbKey, thumpOutputPath, "image/jpg"); err != nil {
		return nil, fmt.Errorf("thumbnail upload failed: %w", err)
	}

	runner = ffmpeg.NewProbeExtractor(ctx, outputPath)
	result, err = runner.Run()
	if err != nil {
		return nil, err
	}

	if !result.IsValid() {
		return nil, fmt.Errorf("Invalid file")
	}

	preResult := processor.PrePrecessedResult{
		VideoKey:     videoKey,
		ThumbnailKey: thumbKey,
		ContentType:  "video/mp4",
		Size:         result.Format.Size,
		Duration:     result.Format.Duration,
	}

	return &preResult, nil
}

func (f *Fmeg) getContentType(ctx context.Context, key string) string {
	info, err := f.minio.Status(ctx, key)
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
