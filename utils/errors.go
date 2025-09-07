package utils

import "errors"

var (
	ErrLoadX509KeyPairFailed = errors.New("failed to load X509 key pair")
	ErrReadCACertFailed      = errors.New("failed to read CA cert")
	ErrAppendCACertFailed    = errors.New("failed to append CA cert")
)
