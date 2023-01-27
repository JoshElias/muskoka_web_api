package muskoka

import (
	"crypto/rand"
	"fmt"
	"regexp"

	"github.com/dgrijalva/jwt-go"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"golang.org/x/crypto/bcrypt"
)

type UserRegistration struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password`
}

type UserVerification struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

type UserCredentials struct {
	ID       string `json:"id" form:"id"`
	Password string `json:"password" form:"password"`
}

func RegisterHandler(ctx context.Context) {

	userRegistration := &UserRegistration{}
	if err := ctx.ReadJSON(userRegistration); err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(map[string]interface{}{"error": "Unable to read user registration"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userRegistration.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(map[string]interface{}{"error": "Unable to hash password"})
		return
	}

	token := randToken()

	user := User{
		Email:             userRegistration.Email,
		Username:          userRegistration.Username,
		PasswordHash:      passwordHash,
		VerificationToken: token,
	}
	user, err = user.Insert()
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	verifyEmail := Email{
		To: []string{
			user.Email,
		},
		From:    "info@muskokacabco.com",
		Subject: "Verify Your Email Address",
		Text: fmt.Sprintf(`
		Hi %[1]s,
		Please follow this link to verify you are the owner of this new email address. 
		https://www.muskokacabco.com/verify?username=%[1]s&token=%[2]s`,
			user.Username, user.VerificationToken),
	}
	err = verifyEmail.Send()
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(map[string]interface{}{"error": err}) //"Unable to send verification email"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(map[string]interface{}{})
}

func randToken() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func VerifyHandler(ctx context.Context) {

	userVerification := &UserVerification{}
	if err := ctx.ReadJSON(userVerification); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read user verification"})
		return
	}

	stmt, err := GetDBConnection().Prepare(`
		UPDATE users
		SET is_verified = TRUE
		WHERE username = $1 AND verification_token = $2 AND is_verified = FALSE`)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to verify user"})
		return
	}

	_, err = stmt.Exec(userVerification.Username, userVerification.Token)
	if err != nil {
		code, errObj := HandleDBError(err)
		ctx.StatusCode(code)
		ctx.JSON(errObj)
		return
	}

	err = stmt.Close()
	if err != nil {
		code, errObj := HandleDBError(err)
		ctx.StatusCode(code)
		ctx.JSON(errObj)
		return
	}

	var isAdmin bool
	err = GetDBConnection().QueryRow(`
			SELECT is_admin
			FROM users 
			WHERE username = $1 AND verification_token = $2`,
		userVerification.Username, userVerification.Token).Scan(&isAdmin)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": userVerification.Username,
		"isAdmin":  isAdmin,
	})
	tokenString, err := token.SignedString(JWT_SECRET)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(map[string]interface{}{"error": "Unable to create user token"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(map[string]interface{}{"token": tokenString})
}

func LoginHandler(ctx context.Context) {
	credentials := &UserCredentials{}
	if err := ctx.ReadJSON(credentials); err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(map[string]interface{}{"error": "Unable to read user credentials"})
		return
	}

	isEmail, err := regexp.MatchString("@", credentials.ID)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(map[string]interface{}{"error": "Unable to read user credentials"})
		return
	}

	user := &User{}
	if isEmail {
		user.Email = credentials.ID
		err = GetDBConnection().QueryRow(`
			SELECT username, password_hash, is_admin
			FROM users 
			WHERE email = $1 AND is_verified = TRUE`, credentials.ID).Scan(&user.Username, &user.PasswordHash, &user.IsAdmin)
	} else {
		user.Username = credentials.ID
		err = GetDBConnection().QueryRow(`
			SELECT email, password_hash, is_admin
			FROM users 
			WHERE username = $1 AND is_verified = TRUE`, credentials.ID).Scan(&user.Email, &user.PasswordHash, &user.IsAdmin)
	}

	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	err = bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(credentials.Password))
	if err != nil {
		ctx.StatusCode(iris.StatusNotFound)
		ctx.JSON(map[string]interface{}{"error": "Password mismatch"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"isAdmin":  user.IsAdmin,
	})
	tokenString, err := token.SignedString(JWT_SECRET)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(map[string]interface{}{"error": "Error creating user token"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(map[string]interface{}{"token": tokenString})
}
