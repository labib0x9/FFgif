package media

import (
	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Update(key string) error {
	var req media.GifResp
	return s.gifRepo.Update(key, req)
}
