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

type GifConverter struct {
	pCmd *exec.Cmd
	gCmd *exec.Cmd
}

func NewGifConverter(
	ctx context.Context,
	inputPath string, outputPath string, palettePath string,
	Width int, FPS int,
	Start float32, End float32,
	Loop string,
) *GifConverter {

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

	return &GifConverter{
		pCmd: newPaletteExec(ctx, &Pfilter),
		gCmd: newGifExec(ctx, &gFilter),
	}
}

func (f *GifConverter) Run() error {
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

type Mp4Converter struct {
	cmd *exec.Cmd
}

func NewMp4Converter(
	ctx context.Context,
	input string,
	output string,
	codecName string,
) *Mp4Converter {
	return &Mp4Converter{
		cmd: newMp4ConverterExec(ctx, input, output, codecName),
	}
}

func (f *Mp4Converter) Run() error {
	if f.cmd == nil {
		return NilPointerErr
	}

	err := f.cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg mp4 converter run failed: %w", err)
	}

	return nil
}

func newMp4ConverterExec(ctx context.Context, inputPath, outputPath string, CodecName string) *exec.Cmd {
	var cmd *exec.Cmd
	if CodecName == "h264" {
		cmd = exec.CommandContext(
			ctx, "ffmpeg",
			"-i", inputPath,
			"-c:v", "copy",
			"-an",
			"-movflags", "+faststart",
			outputPath,
		)
	} else {
		cmd = exec.CommandContext(
			ctx, "ffmpeg",
			"-i", inputPath,
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-an",
			"-movflags", "+faststart",
			outputPath,
		)
	}
	cmd.Stderr = os.Stderr
	return cmd
}

type ThumbGenerator struct {
	cmd *exec.Cmd
}

func NewThumbGenerator(
	ctx context.Context,
	input string,
	output string,
) *ThumbGenerator {
	return &ThumbGenerator{
		cmd: newThumbGeneratorExec(ctx, input, output),
	}
}

func (f *ThumbGenerator) Run() error {
	if f.cmd == nil {
		return NilPointerErr
	}

	err := f.cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg mp4 converter run failed: %w", err)
	}

	return nil
}

func newThumbGeneratorExec(ctx context.Context, inputPath, outputPath string) *exec.Cmd {
	cmd := exec.CommandContext(
		ctx, "ffmpeg",
		"-ss", "00:00:01",
		"-i", inputPath,
		"-vframes", "1",
		"-q:v", "2",
		outputPath,
	)

	cmd.Stderr = os.Stderr
	return cmd
}
