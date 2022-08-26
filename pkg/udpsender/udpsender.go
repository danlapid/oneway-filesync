package udpsender

import (
	"context"
	"fmt"
	"net"
	"oneway-filesync/pkg/structs"

	"github.com/sirupsen/logrus"
)

type UdpSender struct {
	ip    string
	port  int
	input chan *structs.Chunk
}

func worker(ctx context.Context, conf *UdpSender) {
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
		case share := <-conf.input:
			buf, err := share.Encode()
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Path": share.Path,
					"Hash": fmt.Sprintf("%x", share.Hash),
				}).Errorf("Error encoding share: %v", err)
				continue
			}
			conn.Write(buf)
		}
	}
}

func CreateSender(ctx context.Context, ip string, port int, input chan *structs.Chunk, workercount int) {
	conf := UdpSender{
		ip:    ip,
		port:  port,
		input: input,
	}
	for i := 0; i < workercount; i++ {
		go worker(ctx, &conf)
	}
}
