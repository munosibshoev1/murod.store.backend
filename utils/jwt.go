package utils

import (
    "errors"
    "time"

    "github.com/dgrijalva/jwt-go"
)

var jwtKey = []byte("my_secret_key")

type JWTClaim struct {
    ID   string `json:"id"`
    Role string `json:"role"`
    jwt.StandardClaims
}

func GenerateToken(id string, role string) (string, error) {
    expirationTime := time.Now().Add(24 * time.Hour)
    claims := &JWTClaim{
        ID:   id,
        Role: role,
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: expirationTime.Unix(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(jwtKey)
}

func ValidateToken(signedToken string) (*JWTClaim, error) {
    token, err := jwt.ParseWithClaims(
        signedToken,
        &JWTClaim{},
        func(token *jwt.Token) (interface{}, error) {
            return jwtKey, nil
        },
    )
    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*JWTClaim)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }

    return claims, nil
}
