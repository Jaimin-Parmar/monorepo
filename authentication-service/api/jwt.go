package api

import (
	acProtobuf "authentication-service/proto/v1/account"
	"time"

	"github.com/golang-jwt/jwt"
)

type Claims struct {
	UserID   int    `json:"userID"`
	UserName string `json:"userName"`
	jwt.StandardClaims
}

type JWTToken struct {
	Value     string
	ExpiresAt time.Time
}

func createJWTToken(account *acProtobuf.AccountReply, tokenExpiration time.Duration, JWTKey string) (*JWTToken, error) {
	expirationTime := time.Now().Add(tokenExpiration * time.Hour)
	claims := &Claims{
		UserID: int(account.Id),
		// Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Create the JWT string
	tokenString, err := token.SignedString([]byte(JWTKey))
	if err != nil {
		return nil, err
	}
	return &JWTToken{
		Value:     tokenString,
		ExpiresAt: expirationTime,
	}, nil
}
