package media

import (
	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) GetByKey(key string) (media.GifResp, error) {
	return s.gifRepo.GetByKey(key)
}
