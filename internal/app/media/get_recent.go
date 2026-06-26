package media

import (
	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) GetRecents(id string) ([]media.GifResp, error) {
	return s.gifRepo.GetRecents(id)
}
