package ffmpeg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGIFGeneration(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}

	tmpDir := t.TempDir()

	inputPath := filepath.Join("testdata", "sample.mov")
	outputPath := filepath.Join(tmpDir, "output.gif")
	palettePath := filepath.Join(tmpDir, "palette.png")

	ff := NewFFmpeg(
		context.Background(),
		inputPath,
		outputPath,
		palettePath,
		480, // width
		10,  // fps
		0,   // start
		1,   // end
		"0", // loop forever
	)

	if err := ff.Run(); err != nil {
		t.Fatalf("gif generation failed: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output gif not found: %v", err)
	}

	if info.Size() == 0 {
		t.Fatal("generated gif is empty")
	}
}
