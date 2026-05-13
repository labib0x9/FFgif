package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/labib0x9/ProjectUnsafe/repo"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

type Fmeg struct {
	minioRepo repo.UploaderRepository
}

func NewFmeg(minioRepo repo.UploaderRepository) *Fmeg {
	return &Fmeg{
		minioRepo: minioRepo,
	}
}

type PaletteFilter struct {
	Start     float32
	Duration  float32
	InputPath string
	Filter    string
	Path      string
}

type GifFilter struct {
	Start       float32
	Duration    float32
	InputPath   string
	PalettePath string
	Loop        string
	Filter      string
	Path        string
}

func newFork(ctx context.Context, filter any) *exec.Cmd {
	var cmd *exec.Cmd
	if f, ok := filter.(PaletteFilter); ok {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-ss", fmt.Sprintf("%2f", f.Start),
			"-t", fmt.Sprintf("%2f", f.Duration),
			"-i", f.InputPath,
			"-vf", f.Filter,
			"-y",
			f.Path,
		)
	}
	if f, ok := filter.(GifFilter); ok {
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-ss", fmt.Sprintf("%2f", f.Start),
			"-t", fmt.Sprintf("%2f", f.Duration),
			"-i", f.InputPath,
			"-i", f.PalettePath,
			"-filter_complex", f.Filter,
			"-loop", f.Loop,
			"-y",
			f.Path,
		)
	}
	cmd.Stderr = os.Stderr
	return cmd
}

func (f *Fmeg) Process(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (string, error) {
	inputPath := filepath.Join(os.TempDir(), Key+"_input."+f.getContentType(ctx, Key))
	outputPath := filepath.Join(os.TempDir(), Key+"_output.gif")
	palettePath := filepath.Join(os.TempDir(), Key+"_palette.png")

	defer os.Remove(inputPath)
	defer os.Remove(outputPath)
	defer os.Remove(palettePath)

	duration := End - Start

	if err := f.minioRepo.Download(ctx, Key, inputPath); err != nil {
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

	paletteFilter := fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos,palettegen", FPS, Width)
	Pfilter := PaletteFilter{
		Start:     Start,
		Duration:  duration,
		InputPath: inputPath,
		Filter:    paletteFilter,
		Path:      palettePath,
	}

	cmd := newFork(ctx, Pfilter)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("palette generation failed: %w", err)
	}

	gifFilter := fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos[x];[x][1:v]paletteuse", FPS, Width)
	gFilter := GifFilter{
		Start:       Start,
		Duration:    duration,
		InputPath:   inputPath,
		PalettePath: palettePath,
		Loop:        loop,
		Filter:      gifFilter,
		Path:        outputPath,
	}

	cmd = newFork(ctx, gFilter)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gif conversion failed: %w", err)
	}

	gifKey := utils.GenerateRandomID().String() + "_output.gif"
	if err := f.minioRepo.Upload(ctx, gifKey, outputPath, "image/gif"); err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	return gifKey, nil
}

func (f *Fmeg) getContentType(ctx context.Context, key string) string {
	info, err := f.minioRepo.StatObject(ctx, key)
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
