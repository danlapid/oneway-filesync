//go:build linux || darwin

package utils

import (
	"os"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

func sendCtrlC(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	err = p.Signal(os.Interrupt)
	if err != nil {
		return err
	}
	return nil
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
	if runtime.GOOS == "linux" {
		// See https://man7.org/linux/man-pages/man7/socket.7.html SO_RCVBUF
		return bufsize / 2, nil
	} else {
		return bufsize, nil
	}
}
