package main

import (
	"context"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/udpreceiver"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	conf, err := config.GetConfig("config.toml")
	if err != nil {
		logrus.Errorf("Failed reading config with err %v\n", err)
		return

	}

	chunks_chan := make(chan []byte, 20000)

	ctx, cancel := context.WithCancel(context.Background()) // Create a cancelable context and pass it to all goroutines, allows us to gracefully shut down the program

	udpreceiver.CreateReceiver(ctx, conf.ReceiverIP, conf.ReceiverPort, conf.ChunkSize, chunks_chan, 20)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	cancel() // Gracefully shutdown and stop all goroutines
}
