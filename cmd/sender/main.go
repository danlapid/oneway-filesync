package main

import (
	"fmt"
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
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", conf.ReceiverIP, conf.ReceiverPort))
	if err != nil {
		log.Errorf("Some error %v", err)
		return
	}
	defer conn.Close()
	conn.Write([]byte("Message1"))
}
