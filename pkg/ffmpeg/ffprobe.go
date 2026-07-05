package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/labib0x9/ffgif/internal/domain/processor"
)

type ProbeExtractor struct {
	cmd *exec.Cmd
}

func NewProbeExtractor(
	ctx context.Context,
	inputPath string,
) *ProbeExtractor {
	return &ProbeExtractor{
		cmd: newProbeExex(ctx, inputPath),
	}
}

func (f *ProbeExtractor) Run() (*processor.FFprobeOutput, error) {
	if f.cmd == nil {
		return nil, NilPointerErr
	}

	out, err := f.cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe run failed: %w", err)
	}

	var res processor.FFprobeOutput
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("ffprobe unmarshal failed: %w", err)
	}

	return &res, nil
}

func newProbeExex(ctx context.Context, input string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-select_streams", "v:0",
		input,
	)
	cmd.Stderr = os.Stderr
	return cmd
}
