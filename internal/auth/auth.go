package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("my_secret_claim")

type Claims struct {
	UserID				string 	`json:"userId"`
	Username 			string 	`json:"username"`
	Email 				string 	`json:"email"`
	RefreshTokenVersion int 	`json:"refreshTokenVersion"`
	jwt.RegisteredClaims
}

func ValidateJWT(reqToken string) (*Claims, error)  {
	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(reqToken, claims, func(token *jwt.Token) (any, error) {
		return jwtKey, nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, err
		}
		return nil, fmt.Errorf("bad request")
	}
	if !tkn.Valid {
		return nil, fmt.Errorf("unauthorized")
	}

	return claims, nil
}

func generateJWT(userID string) (string, error) {
	expirationTime := time.Now().Add(15 * time.Minute)
	claims := &Claims{
		UserID: userID,
		Username: "username",
		Email: "",
		RefreshTokenVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(jwtKey)
}

func issueNewToken(claims *Claims) (string, error) {
	expirationTime := time.Now().Add(15 * time.Minute)
	newClaims := &Claims{
		UserID: claims.UserID,
		Username: claims.Username,
		Email: claims.Email,
		RefreshTokenVersion: claims.RefreshTokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, newClaims)
	return token.SignedString(jwtKey)
}
