package udpreceiver

import (
	"context"
	"net"
	"oneway-filesync/pkg/structs"

	"github.com/sirupsen/logrus"
)

type UdpReceiver struct {
	conn      *net.UDPConn
	chunksize int
	output    chan structs.Chunk
}

func Worker(ctx context.Context, conf UdpReceiver) {
	buf := make([]byte, conf.chunksize)

	for {
		// conn.Close will interrupt any waiting ReadFromUDP
		select {
		case <-ctx.Done():
			return
		default:
			n, _, err := conf.conn.ReadFromUDP(buf)
			if err != nil {
				logrus.Errorf("Error reading from socket:  %v", err)
				continue
			}
			chunk, err := structs.DecodeChunk(buf[:n])
			if err != nil {
				logrus.Errorf("Error decoding chunk:  %v", err)
				continue
			}
			conf.output <- chunk
		}
	}
}

func CreateReceiver(ctx context.Context, ip string, port int, chunksize int, output chan structs.Chunk, workercount int) {
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
		go Worker(ctx, conf)
	}
}
