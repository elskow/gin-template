package helpers

import (
	"github.com/elskow/go-microservice-template/config"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	cfg := config.Get()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cfg.BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(plainPassword string, hashPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(plainPassword))
	return err == nil
}
