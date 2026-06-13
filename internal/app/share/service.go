package share

type Service interface {
	Create()
	Delete()
	Download()
	Get()
	Update()
	View()
}

type service struct {
}

func NewService() Service {
	return &service{}
}

// type Jwt interface {
// 	Create(fullname string, id string, email string, role string) (string, error)
// 	Verify(tokenStr string) (jwt.Payload, error)
// }

// type Hasher interface {
// 	GenerateHash(pass string) (string, error)
// 	CompareHashAndPassword(hashedPass string, pass string) bool
// }
