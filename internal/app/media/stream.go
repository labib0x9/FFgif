package media

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

type StreamResult struct {
	Start int64
	End   int64
	Obj   media.Object
	Info  media.Info
}

func (s *service) Stream(ctx context.Context, key string, Range string) (*StreamResult, error) {
	info, err := s.storage.Status(ctx, key)
	if err != nil {
		// slog.Info("Stream: info missed")
		// http.Error(w, "not found", http.StatusNotFound)
		return nil, media.ErrStatFetchFailed
	}

	start, end, err := parseRange(Range, info.Size)
	if err != nil {
		// slog.Info("Stream", "err", err)
		// http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
		return nil, media.ErrRangeParserFailed
	}

	obj, err := s.storage.GetObject(ctx, start, end, key)
	if err != nil {
		// http.Error(w, "stream failed", http.StatusInternalServerError)
		return nil, media.ErrObjectFetchFailed
	}

	return &StreamResult{
		Obj:   obj,
		Start: start,
		End:   end,
		Info:  info,
	}, nil
}

func parseRange(r string, total int64) (start int64, end int64, err error) {

	if r == "" {
		return 0, total - 1, nil
	}

	if !strings.HasPrefix(r, "bytes=") {
		return 0, 0, errors.New("invalid range")
	}

	parts := strings.Split(strings.TrimPrefix(r, "bytes="), "-")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid range")
	}

	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	if parts[1] == "" {
		end = total - 1
	} else {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
	}

	if end >= total {
		end = total - 1
	}

	if start > end {
		return 0, 0, errors.New("range out of bounds")
	}

	return
}
