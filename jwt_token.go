package main

import (
	"github.com/dgrijalva/jwt-go"
)

type jwtTokenGen struct {
	publicKey  []byte
	privateKey []byte
}

func (j *jwtTokenGen) Generate(claims jwt.Claims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok, err := token.SignedString(j.privateKey)
	if err != nil {
		panic(err)
	}
	return tok
}
