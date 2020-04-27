package database

import (
	"github.com/rishabh-bector/BenevolentBitesBack/auth"
)

// ValidateUser authorizes incoming frontend requests through the user's JWT
func ValidateUser(token string) string {
	claims, err := auth.ValidateToken(token)
	if claims == nil || err != nil {
		// Invalid token
		return "nil"
	}
	return claims["email"].(string)
}
