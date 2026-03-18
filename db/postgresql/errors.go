package postgresql

import "errors"

var (
	ErrInvalidSteps           = errors.New("invalid steps")
	ErrHostnameRequired       = errors.New("hostname is required")
	ErrPortRequired           = errors.New("port is required")
	ErrUsernameRequired       = errors.New("username is required")
	ErrPasswordRequired       = errors.New("password is required")
	ErrDatabaseRequired       = errors.New("database is required")
	ErrMigrationsPathRequired = errors.New("migrations path is required")
)
