//go:build windows

package utils

import (
	"errors"
	"syscall"
)

func GetReadBuffer(rawconn syscall.RawConn) (int, error) {
	return 0, errors.New("unsupported OS")
}

func GetAvailableBytes(rawconn syscall.RawConn) (int, error) {
	return 0, errors.New("unsupported OS")
}
