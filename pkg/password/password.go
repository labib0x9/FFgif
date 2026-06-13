package password

import "golang.org/x/crypto/bcrypt"

type Hasher struct {
	Pepper string
	Cost   int
}

func NewHasher(pepper string, cost int) *Hasher {
	return &Hasher{
		Pepper: pepper,
		Cost:   cost,
	}
}

func (h *Hasher) GenerateHash(pass string) (string, error) {
	pass += h.Pepper
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), h.Cost)
	return string(hash), err
}

func (h *Hasher) CompareHashAndPassword(hashedPass string, pass string) bool {
	pass += h.Pepper
	err := bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(pass))
	return err == nil
}
