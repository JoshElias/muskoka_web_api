package muskoka

import (
	"sync"
	jwt "github.com/dgrijalva/jwt-go"
	jwtmiddleware "github.com/iris-contrib/middleware/jwt"
)

var JWT_SECRET = []byte("secret")


var jwtMiddleware *jwtmiddleware.Middleware
var jwtOnce sync.Once


func JWTMiddleware() *jwtmiddleware.Middleware {
	jwtOnce.Do(func() {
		jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Config{
			ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
				return JWT_SECRET, nil
			},
			SigningMethod: jwt.SigningMethodHS256,
		})
	})
	return jwtMiddleware
}
