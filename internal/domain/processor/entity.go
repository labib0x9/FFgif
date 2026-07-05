package processor

import "strconv"

type Format struct {
	Filename   string `json:"filename"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	ProbeScore int    `json:"probe_score"`
}

type Stream struct {
	CodecName       string `json:"codec_name"`
	CodecType       string `json:"codec_type"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	MimeCodecString string `json:"mime_codec_string"`
}

// only the first stream, stream[0]
type FFprobeOutput struct {
	Format Format   `json:"format"`
	Stream []Stream `json:"streams"`
}

func (f *FFprobeOutput) IsValid() bool {
	if f.Format.ProbeScore < 50 {
		return false
	}
	size, err := strconv.ParseInt(f.Format.Size, 10, 32)
	if err != nil {
		return false
	}
	if size > 500*1024*1024 {
		return false
	}
	if f.Stream[0].CodecType != "video" {
		return false
	}
	return true
}

type PrePrecessedResult struct {
	VideoKey     string
	ThumbnailKey string
	Size         string
	Duration     string
	ContentType  string
}
