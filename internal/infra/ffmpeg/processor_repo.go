package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/port/processor"
	"github.com/labib0x9/ffgif/pkg/random"
)

type fmeg struct {
	minio media.StorageRepository
}

func NewFmeg(minioRepo media.StorageRepository) processor.VideoProcessor {
	return &fmeg{
		minio: minioRepo,
	}
}

func (f *fmeg) Process(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*processor.JobResult, error) {
	inputPath := filepath.Join(os.TempDir(), "input_"+Key+".mp4")
	outputPath := filepath.Join(os.TempDir(), Key+"_output.gif")
	palettePath := filepath.Join(os.TempDir(), Key+"_palette.png")
	thumpOutputPath := filepath.Join(os.TempDir(), "thumb_"+Key+".jpg")

	defer os.Remove(inputPath)
	defer os.Remove(outputPath)
	defer os.Remove(palettePath)
	defer os.Remove(thumpOutputPath)

	if err := f.minio.DownloadLocal(ctx, Key, inputPath); err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
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

	runner := NewGifConverter(ctx, inputPath, outputPath, palettePath, Width, FPS, Start, End, loop)
	if err := runner.Run(); err != nil {
		return nil, fmt.Errorf("gif conversion failed: %w", err)
	}

	videoThumbGenerator := NewThumbGenerator(ctx, inputPath, thumpOutputPath, Start+1.0)
	if err := videoThumbGenerator.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed to generate thumbnail from video: %w", err)
	}

	gifKey := random.GenerateRandomID().String() + "_output.gif"
	if err := f.minio.Upload(ctx, gifKey, outputPath, "image/gif"); err != nil {
		return nil, fmt.Errorf("gif upload failed: %w", err)
	}

	thumbKey := "thumpnail_" + random.GenerateRandomID().String() + ".jpg"
	if err := f.minio.Upload(ctx, thumbKey, thumpOutputPath, "image/jpg"); err != nil {
		return nil, fmt.Errorf("thumbnail upload failed: %w", err)
	}

	return &processor.JobResult{
		GifKey:   gifKey,
		ThumbKey: thumbKey,
	}, nil
}

// download to local, validate the video file, convert to mp4 and returns the output path
// caller should delete the output video file
func (f *fmeg) PreProcess(ctx context.Context, key string) (*processor.PrePrecessedResult, error) {
	inputPath := filepath.Join(os.TempDir(), "raw_video_"+key)
	outputPath := filepath.Join(os.TempDir(), "converted_"+key+".mp4")
	thumpOutputPath := filepath.Join(os.TempDir(), "thumb_"+key+".jpg")
	defer os.Remove(inputPath)
	defer os.Remove(outputPath)
	defer os.Remove(thumpOutputPath)

	if err := f.minio.DownloadLocalRawVideo(ctx, key, inputPath); err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	runner := NewProbeExtractor(ctx, inputPath)
	result, err := runner.Run()
	if err != nil {
		return nil, err
	}

	if !result.IsValid() {
		return nil, fmt.Errorf("Invalid file")
	}

	mp4Converter := NewMp4Converter(ctx, inputPath, outputPath, result.Stream[0].CodecName)
	if err := mp4Converter.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed to convert raw video to mp4: %w", err)
	}

	videoThumbGenerator := NewThumbGenerator(ctx, outputPath, thumpOutputPath, 1.0)
	if err := videoThumbGenerator.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed to generate thumbnail from raw video: %w", err)
	}

	videoKey := "converted_" + random.GenerateRandomID().String() + ".mp4"
	if err := f.minio.Upload(ctx, videoKey, outputPath, "video/mp4"); err != nil {
		return nil, fmt.Errorf("converted video upload failed: %w", err)
	}

	thumbKey := "thumpnail_" + random.GenerateRandomID().String() + ".jpg"
	if err := f.minio.Upload(ctx, thumbKey, thumpOutputPath, "image/jpg"); err != nil {
		return nil, fmt.Errorf("thumbnail upload failed: %w", err)
	}

	runner = NewProbeExtractor(ctx, outputPath)
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
