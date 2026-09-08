package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/labib0x9/ffgif/pkg/telemetry"
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

	// paletteFilter := fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos,palettegen", FPS, Width)
	paletteFilter := fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos,format=rgb24,palettegen", FPS, Width)
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
	start := time.Now()
	if err := f.pCmd.Run(); err != nil {
		telemetry.MediaConversionsTotal.WithLabelValues("palette", "failure").Inc()
		return fmt.Errorf("palette generation failed: %w", err)
	}
	if err := f.gCmd.Run(); err != nil {
		telemetry.MediaConversionsTotal.WithLabelValues("gif", "failure").Inc()
		return fmt.Errorf("gif conversion failed: %w", err)
	}
	duration := time.Since(start).Seconds()
	telemetry.MediaConversionDuration.WithLabelValues("gif").Observe(duration)
	telemetry.MediaConversionsTotal.WithLabelValues("gif", "success").Inc()
	return nil
}

func newPaletteExec(ctx context.Context, f *PaletteFilter) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", fmt.Sprintf("%.2f", f.Start),
		"-t", fmt.Sprintf("%.2f", f.Duration),
		"-i", f.InputPath,
		"-vf", f.Filter,
		"-update", "1",
		"-frames:v", "1",
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

	start := time.Now()
	err := f.cmd.Run()
	if err != nil {
		telemetry.MediaConversionsTotal.WithLabelValues("mp4", "failure").Inc()
		return fmt.Errorf("ffmpeg mp4 converter run failed: %w", err)
	}

	duration := time.Since(start).Seconds()
	telemetry.MediaConversionDuration.WithLabelValues("mp4").Observe(duration)
	telemetry.MediaConversionsTotal.WithLabelValues("mp4", "success").Inc()

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
			"-y",
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
			"-y",
			outputPath,
		)
	}
	cmd.Stderr = os.Stderr
	return cmd
}

type ThumbGenerator struct {
	cmd *exec.Cmd
}

// startAt = in seconds from video
func NewThumbGenerator(
	ctx context.Context,
	input string,
	output string,
	startAt float32,
) *ThumbGenerator {
	return &ThumbGenerator{
		cmd: newThumbGeneratorExec(ctx, startAt, input, output),
	}
}

func (f *ThumbGenerator) Run() error {
	if f.cmd == nil {
		return NilPointerErr
	}

	start := time.Now()
	err := f.cmd.Run()
	if err != nil {
		telemetry.MediaConversionsTotal.WithLabelValues("thumbnail", "failure").Inc()
		return fmt.Errorf("ffmpeg mp4 converter run failed: %w", err)
	}

	duration := time.Since(start).Seconds()
	telemetry.MediaConversionDuration.WithLabelValues("thumbnail").Observe(duration)
	telemetry.MediaConversionsTotal.WithLabelValues("thumbnail", "success").Inc()

	return nil
}

func newThumbGeneratorExec(ctx context.Context, startAt float32, inputPath, outputPath string) *exec.Cmd {
	cmd := exec.CommandContext(
		ctx, "ffmpeg",
		"-ss", fmt.Sprintf("%.2f", startAt),
		"-i", inputPath,
		"-vframes", "1",
		"-q:v", "2",
		"-update", "1",
		"-vf", "format=yuvj420p",
		outputPath,
	)

	cmd.Stderr = os.Stderr
	return cmd
}
