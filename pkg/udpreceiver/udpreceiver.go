package udpreceiver

import (
	"context"
	"net"

	"github.com/sirupsen/logrus"
)

type UdpReceiver struct {
	ip        string
	port      int
	chunksize int
	output    chan []byte
}

func Worker(ctx context.Context, conf UdpReceiver) {
	buf := make([]byte, conf.chunksize)
	addr := net.UDPAddr{
		IP:   net.ParseIP(conf.ip),
		Port: conf.port,
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

	for {
		// conn.Close will interrupt any waiting ReadFromUDP
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			logrus.Errorf("Some error  %v", err)
			continue
		}
		conf.output <- buf[:n]
	}
}

func CreateReceiver(ctx context.Context, ip string, port int, chunksize int, output chan []byte, workercount int) {
	conf := UdpReceiver{
		ip:        ip,
		port:      port,
		chunksize: chunksize,
		output:    output,
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, conf)
	}
}
