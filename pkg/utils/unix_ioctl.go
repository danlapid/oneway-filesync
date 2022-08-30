//go:build linux || darwin

package utils

import (
	"errors"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

func sendCtrlC(int pid) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	err = p.Signal(os.Interrupt)
	if err != nil {
		return err
	}
}

func GetReadBuffer(rawconn syscall.RawConn) (int, error) {
	var err error
	var bufsize int
	err2 := rawconn.Control(func(fd uintptr) {
		bufsize, err = unix.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	})
	if err2 != nil {
		return 0, err2
	}
	if err != nil {
		return 0, err
	}
	return bufsize, nil
}

func GetAvailableBytes(rawconn syscall.RawConn) (int, error) {
	var FIONREAD uint = 0
	if runtime.GOOS == "linux" {
		FIONREAD = 0x541B
	} else if runtime.GOOS == "darwin" {
		FIONREAD = 0x4004667f
	} else {
		return 0, errors.New("unsupported OS")
	}

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
