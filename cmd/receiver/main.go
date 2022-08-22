package main

import (
	"net"
	"oneway-filesync/pkg/config"

	"github.com/sirupsen/logrus"
)

func main() {
	conf, err := config.GetConfig("config.toml")
	if err != nil {
		logrus.Errorf("Failed reading config with err %v\n", err)
		return

	}

	buf := make([]byte, conf.ChunkSize)
	addr := net.UDPAddr{
		Port: conf.ReceiverPort,
		IP:   net.ParseIP(conf.ReceiverIP),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		logrus.Errorf("Some error %v\n", err)
		return
	}

	for {
		_, remoteaddr, err := conn.ReadFromUDP(buf)
		logrus.Infof("Read a message from %v %d \n", remoteaddr, len(buf))
		if err != nil {
			logrus.Errorf("Some error  %v", err)
			continue
		}
	}
}

func GetConfig(s string) {
	panic("unimplemented")
}
