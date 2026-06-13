package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

var (
	NilPointerErr = errors.New("exec cmd nil pointer")
)

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

type Ffmpeg struct {
	pCmd *exec.Cmd
	gCmd *exec.Cmd
}

func NewFFmpeg(
	ctx context.Context,
	inputPath string, outputPath string, palettePath string,
	Width int, FPS int,
	Start float32, End float32,
	Loop string,
) *Ffmpeg {

	paletteFilter := fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos,palettegen", FPS, Width)
	Pfilter := PaletteFilter{
		Start:     Start,
		Duration:  End - Start,
		InputPath: inputPath,
		Filter:    paletteFilter,
		Path:      palettePath,
	}

	gifFilter := fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos[x];[x][1:v]paletteuse", FPS, Width)
	gFilter := GifFilter{
		Start:       Start,
		Duration:    End - Start,
		InputPath:   inputPath,
		PalettePath: palettePath,
		Loop:        Loop,
		Filter:      gifFilter,
		Path:        outputPath,
	}

	return &Ffmpeg{
		pCmd: newPaletteExec(ctx, &Pfilter),
		gCmd: newGifExec(ctx, &gFilter),
	}
}

func (f *Ffmpeg) Run() error {
	if f.pCmd == nil || f.gCmd == nil {
		return NilPointerErr
	}
	if err := f.pCmd.Run(); err != nil {
		return fmt.Errorf("palette generation failed: %w", err)
	}
	if err := f.gCmd.Run(); err != nil {
		return fmt.Errorf("gif conversion failed: %w", err)
	}
	return nil
}

func newPaletteExec(ctx context.Context, f *PaletteFilter) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", fmt.Sprintf("%.2f", f.Start),
		"-t", fmt.Sprintf("%.2f", f.Duration),
		"-i", f.InputPath,
		"-vf", f.Filter,
		"-y",
		f.Path,
	)
	cmd.Stderr = os.Stderr
	return cmd
}

func newGifExec(ctx context.Context, f *GifFilter) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", fmt.Sprintf("%.2f", f.Start),
		"-t", fmt.Sprintf("%.2f", f.Duration),
		"-i", f.InputPath,
		"-i", f.PalettePath,
		"-filter_complex", f.Filter,
		"-loop", f.Loop,
		"-y",
		f.Path,
	)
	cmd.Stderr = os.Stderr
	return cmd
}
