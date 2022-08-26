package udpreceiver

import (
	"context"
	"errors"
	"net"
	"oneway-filesync/pkg/structs"
	"oneway-filesync/pkg/utils"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

type UdpReceiver struct {
	conn      *net.UDPConn
	chunksize int
	output    chan *structs.Chunk
}

func manager(ctx context.Context, conf *UdpReceiver) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		logrus.Infof("Buffers fill detection not supported on the current OS")
		return
	}

	ticker := time.NewTicker(200 * time.Millisecond)
	rawconn, err := conf.conn.SyscallConn()
	if err != nil {
		logrus.Errorf("Error getting raw socket: %v", err)
		return
	}
	bufsize, err := utils.GetReadBuffer(rawconn)
	if err != nil {
		logrus.Errorf("Error getting read buffer size: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			toread, err := utils.GetAvailableBytes(rawconn)
			if err != nil {
				logrus.Errorf("Error getting available bytes on socket: %v", err)
			}

			if float64(toread)/float64(bufsize) > 0.8 {
				logrus.Errorf("Buffers are filling up loss of data is probable")
			}
		}
	}
}

func worker(ctx context.Context, conf *UdpReceiver) {
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
		go worker(ctx, &conf)
	}
	go manager(ctx, &conf)
}
