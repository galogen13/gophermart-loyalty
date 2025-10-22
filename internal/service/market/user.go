package market

import "errors"

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	ID       *int64 `json:"-"`
}

var ErrUserLoginAlreadyInUse error = errors.New("login already in use")
var ErrUserNotExists error = errors.New("user not exists")
