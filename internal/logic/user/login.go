package user

import (
	"log"

	"github.com/google/uuid"
)

func (u *User) Login(username string) (token string, err error) {
	// 生成 JWT 令牌
	uID := uuid.New()
	token, err = u.jwt.GenerateAccessToken(uID.String(), username)
	if err != nil {
		log.Printf("GenerateAccessToken failed: %v", err)
		return "", err
	}
	return token, nil
}
