package udpsender

import (
	"context"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
)

type UdpSender struct {
	ip    string
	port  int
	input chan []byte
}

func Worker(ctx context.Context, conf UdpSender) {
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", conf.ip, conf.port))
	if err != nil {
		logrus.Errorf("Error creating udp socket: %v", err)
		return
	}
	defer conn.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case buf := <-conf.input:
			conn.Write(buf)
		}
	}
}

func CreateSender(ctx context.Context, ip string, port int, input chan []byte, workercount int) {
	conf := UdpSender{
		ip:    ip,
		port:  port,
		input: input,
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, conf)
	}
}
