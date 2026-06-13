package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	EmptyValueErr = errors.New("field empty")
)

type Payload struct {
	Fullname string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type Jwt struct {
	JwtSecretKey []byte
}

func NewJwt(key []byte) *Jwt {
	return &Jwt{
		JwtSecretKey: key,
	}
}

func (j *Jwt) Create(fullname string, id string, email string, role string) (string, error) {
	var claims Payload
	if fullname == "" || email == "" || role == "" {
		return "", EmptyValueErr
	}
	if role != "anon" {
		claims = Payload{
			Fullname: fullname,
			Email:    email,
			Role:     role,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   id,
				Issuer:    "projectpdf",
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			},
		}
	} else {
		claims = Payload{
			Fullname: fullname,
			Role:     role,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   id,
				Issuer:    "projectpdf",
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			},
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.JwtSecretKey)
}

func (j *Jwt) Verify(tokenStr string) (Payload, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Payload{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("wrong signing method: %v", t.Header["alg"])
			}
			return j.JwtSecretKey, nil
		},
	)

	if err != nil {
		return Payload{}, err
	}

	claims, ok := token.Claims.(*Payload)
	if !ok {
		return Payload{}, errors.New("invalid claims")
	}

	return *claims, nil
}
