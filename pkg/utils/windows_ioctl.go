//go:build windows

package utils

import (
	"syscall"
	"unsafe"
)

var (
	ws2_32      = syscall.NewLazyDLL("ws2_32.dll")
	ioctlsocket = ws2_32.NewProc("ioctlsocket")
	// kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	// generateConsoleCtrlEvent = kernel32.NewProc("GenerateConsoleCtrlEvent")
)

const FIONREAD int32 = 0x4004667f

func ioctlSocket(s syscall.Handle, cmd int32) (int, error) {
	v := uint32(0)
	rc, _, err := ioctlsocket.Call(uintptr(s), uintptr(cmd), uintptr(unsafe.Pointer(&v)))
	if rc == 0 {
		return int(v), nil
	} else {
		return 0, err
	}
}

func getsockoptInt(fd syscall.Handle, level, opt int) (int, error) {
	v := int32(0)
	l := int32(unsafe.Sizeof(v))
	err := syscall.Getsockopt(fd, int32(level), int32(opt), (*byte)(unsafe.Pointer(&v)), &l)
	return int(v), err
}

// Removed CtrlC test due to: https://github.com/golang/go/issues/46354
// func sendCtrlC(pid int) error {
// 	r, _, e := generateConsoleCtrlEvent.Call(syscall.CTRL_C_EVENT, uintptr(pid))
// 	if r == 0 {
// 		return e
// 	} else {
// 		return nil
// 	}
// }

func GetReadBuffer(rawconn syscall.RawConn) (int, error) {
	var err error
	var bufsize int
	err2 := rawconn.Control(func(fd uintptr) {
		bufsize, err = getsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF)
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
	var err error
	var avail int
	err2 := rawconn.Control(func(fd uintptr) {
		avail, err = ioctlSocket(syscall.Handle(fd), FIONREAD)
	})
	if err2 != nil {
		return 0, err2
	}
	if err != nil {
		return 0, err
	}
	return avail, nil

}
