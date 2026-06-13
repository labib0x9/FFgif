package media

import (
	"github.com/labib0x9/ProjectUnsafe/internal/domain/media"
)

func (s *service) GetRecents(id string) ([]media.GifResp, error) {
	return s.gifRepo.GetRecents(id)
}
