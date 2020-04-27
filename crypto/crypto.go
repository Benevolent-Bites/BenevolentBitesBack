package crypto

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

var key []byte

func Initialize() {
	key = []byte(os.Getenv("S_SECRET"))
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func SignString(s string) (string, error) {
	claims := jwt.StandardClaims{
		ExpiresAt: time.Now().Unix() + 3600,
		Subject:   s,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
}

func ValidateJWT(s string) (string, error) {
	token, err := jwt.Parse(s, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if token.Valid {
			return claims["sub"].(string), nil
		}
		return "", errors.New("card id expired, please regenerate")
	}
	return "", fmt.Errorf("invalid card claims")
}
