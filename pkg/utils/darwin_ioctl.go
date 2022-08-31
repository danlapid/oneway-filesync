//go:build darwin

package utils

import (
	"syscall"

	"golang.org/x/sys/unix"
)

const FIONREAD uint = 0x4004667f

func GetAvailableBytes(rawconn syscall.RawConn) (int, error) {
	var err error
	var avail int
	err2 := rawconn.Control(func(fd uintptr) {
		avail, err = unix.IoctlGetInt(int(fd), FIONREAD)
	})
	if err2 != nil {
		return 0, err2
	}
	if err != nil {
		return 0, err
	}
	return avail, nil
}
