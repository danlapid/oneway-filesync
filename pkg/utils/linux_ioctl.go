//go:build linux

package utils

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// Under linux FIONREAD returns the size of the waiting datagram if one exists and not the total available bytes
// See: https://manpages.debian.org/bullseye/manpages/udp.7.en.html#FIONREAD
// Sadly the only way to get the available bytes under linux is through proc/udp
func GetAvailableBytes(rawconn syscall.RawConn) (int, error) {
	var err error
	var link string
	err2 := rawconn.Control(func(fd uintptr) {
		link, err = os.Readlink(fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), int(fd)))
	})
	if err2 != nil {
		return 0, err2
	}
	if err != nil {
		return 0, err
	}

	parts := strings.Split(link, ":[")
	if parts[0] != "socket" {
		return 0, errors.New("failed parsing /proc/<pid>/fd/<sock> link")
	}

	inode, err := strconv.ParseUint(parts[1][:len(parts[1])-1], 0, 64)
	if err != nil {
		return 0, err
	}

	netudp, err := GetNetUDP()
	if err != nil {
		return 0, err
	}
	for _, l := range netudp {
		if l.Inode == inode {
			// The division by 2 is due to the same overehead mentioned in SO_RCVBUF
			return int(l.RxQueue / 2), nil
		}
	}
	return 0, errors.New("socket inode was not found in proc/net/udp")
}
