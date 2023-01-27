package muskoka

import (
//"log"
)

type User struct {
	ID                int64  `json:"id"`
	Email             string `json:"email"`
	Username          string `json:"username"`
	PasswordHash      []byte `json:"passwordHash"`
	IsAdmin           bool   `json:"isAdmin"`
	IsVerified        bool   `json:"isVerified"`
	VerificationToken string `json:"verificationToken"`
}

func InitUser() {
	createUserTable()
	createUserIndices()
}

func createUserTable() {
	_, err := GetDBConnection().Exec(`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			email text NOT NULL,
			username text NOT NULL,
			password_hash bytea NOT NULL,
			is_admin BOOLEAN NOT NULL DEFAULT FALSE,
			is_verified BOOLEAN NOT NULL DEFAULT FALSE,
			verification_token character varying(100) NOT NULL
		);`)
	if err != nil {
		panic(err)
	}
}

func createUserIndices() {
	_, err := GetDBConnection().Exec(`CREATE UNIQUE INDEX IF NOT EXISTS users__email__key ON users (lower(email));`)
	if err != nil {
		panic(err)
	}

	_, err = GetDBConnection().Exec(`CREATE UNIQUE INDEX IF NOT EXISTS users__username__key ON users (lower(username));`)
	if err != nil {
		panic(err)
	}
}

func (u User) Insert() (User, error) {
	err := GetDBConnection().QueryRow(`INSERT INTO users (
		email, username, password_hash, is_admin, is_verified, verification_token)
		VALUES($1,$2,$3,$4,$5,$6) returning id;`, u.Email, u.Username, u.PasswordHash,
		u.IsAdmin, u.IsVerified, u.VerificationToken).Scan(&u.ID)
	return u, err
}
