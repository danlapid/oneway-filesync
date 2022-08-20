package main

import (
	"net"
	"oneway-filesync/pkg/config"

	log "github.com/sirupsen/logrus"
)

func main() {
	conf, err := config.GetConfig("config.toml")
	if err != nil {
		log.Errorf("Failed reading config with err %v\n", err)
		return

	}

	buf := make([]byte, conf.BufferSize)
	addr := net.UDPAddr{
		Port: conf.ReceiverPort,
		IP:   net.ParseIP(conf.ReceiverIP),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Errorf("Some error %v\n", err)
		return
	}

	for {
		_, remoteaddr, err := conn.ReadFromUDP(buf)
		log.Infof("Read a message from %v %s \n", remoteaddr, buf)
		if err != nil {
			log.Errorf("Some error  %v", err)
			continue
		}
	}
}

func GetConfig(s string) {
	panic("unimplemented")
}
