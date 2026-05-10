package ffmpeg

import "context"

type Fmeg struct {
}

func NewFmeg() *Fmeg {
	return &Fmeg{}
}

func (f *Fmeg) Process(ctx context.Context, videoID, inputPath, outputPath string, formats []string) error {
	return nil
}
