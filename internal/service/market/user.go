package market

import "errors"

type UserCredentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type User struct {
	UserCredentials
	ID *int64 `json:"-"`
}

var ErrUserLoginAlreadyInUse error = errors.New("login already in use")
var ErrUserNotExists error = errors.New("user not exists")

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
