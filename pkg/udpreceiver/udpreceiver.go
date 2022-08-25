package udpreceiver

import (
	"context"
	"errors"
	"net"
	"oneway-filesync/pkg/structs"
	"runtime"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type UdpReceiver struct {
	conn      *net.UDPConn
	chunksize int
	output    chan *structs.Chunk
}

func Manager(ctx context.Context, conf *UdpReceiver) {
	var FIONREAD uint = 0
	if runtime.GOOS == "linux" {
		FIONREAD = 0x541B
	} else if runtime.GOOS == "darwin" {
		FIONREAD = 0x4004667f
	} else {
		logrus.Infof("Buffers fill detection not supported on the current OS")
		return
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	rawconn, err := conf.conn.SyscallConn()
	if err != nil {
		logrus.Errorf("Error getting raw socket: %v", err)
		return
	}
	var bufsize int
	err2 := rawconn.Control(func(fd uintptr) {
		bufsize, err = unix.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF)
	})
	if err2 != nil {
		logrus.Errorf("Error running Control for FIONREAD: %v", err2)
	}
	if err != nil {
		logrus.Errorf("Error getting FIONREAD: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var toread int = 0
			err2 := rawconn.Control(func(fd uintptr) {
				toread, err = unix.IoctlGetInt(int(fd), FIONREAD)
			})
			if err2 != nil {
				logrus.Errorf("Error running Control for FIONREAD: %v", err2)
			}
			if err != nil {
				logrus.Errorf("Error getting FIONREAD: %v", err)
			}

			if float64(toread)/float64(bufsize) > 0.8 {
				logrus.Errorf("Buffers are filling up loss of data is probable")
			}
		}
	}
}

func Worker(ctx context.Context, conf *UdpReceiver) {
	buf := make([]byte, conf.chunksize)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// conn.Close will interrupt any waiting ReadFromUDP
			n, _, err := conf.conn.ReadFromUDP(buf)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					// conn.Close was called
					continue
				}
				logrus.Errorf("Error reading from socket: %v", err)
				continue
			}
			chunk, err := structs.DecodeChunk(buf[:n])
			if err != nil {
				logrus.Errorf("Error decoding chunk: %v", err)
				continue
			}
			conf.output <- &chunk
		}
	}
}

func CreateUdpReceiver(ctx context.Context, ip string, port int, chunksize int, output chan *structs.Chunk, workercount int) {
	addr := net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		logrus.Errorf("Error creating udp socket: %v", err)
		return
	}
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	conf := UdpReceiver{
		conn:      conn,
		chunksize: chunksize,
		output:    output,
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, &conf)
	}
	go Manager(ctx, &conf)
}
