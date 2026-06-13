package random

import (
	"github.com/google/uuid"
)

func GenerateRandomID() uuid.UUID {
	id, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	return id
}
