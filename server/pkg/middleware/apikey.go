package middleware

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func GenerateSecureAPIKey() string {
	newKey := uuid.New().String()
	return newKey
}

func HashAPIKey(apiKey string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

func CheckAPIKeyHash(apiKey, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
	return err == nil
}
