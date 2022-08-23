package main

import (
	"context"
	"oneway-filesync/pkg/bandwidthlimiter"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/fecencoder"
	"oneway-filesync/pkg/filereader"
	"oneway-filesync/pkg/queuereader"
	"oneway-filesync/pkg/structs"
	"oneway-filesync/pkg/udpsender"
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

	queue_chan := make(chan database.File, 100)
	chunks_chan := make(chan structs.Chunk, 10000)
	shares_chan := make(chan structs.Chunk, 10000)
	bw_limited_chunks := make(chan structs.Chunk, 5) // Small buffer to reduce burst

	ctx, cancel := context.WithCancel(context.Background()) // Create a cancelable context and pass it to all goroutines, allows us to gracefully shut down the program

	queuereader.CreateQueueReader(ctx, queue_chan)
	filereader.CreateFileReader(ctx, conf.ChunkSize, conf.ChunkFecRequired, queue_chan, chunks_chan, 20)
	fecencoder.CreateFecEncoder(ctx, conf.ChunkSize, conf.ChunkFecRequired, conf.ChunkFecTotal, chunks_chan, shares_chan, 20)
	bandwidthlimiter.CreateBandwidthLimiter(ctx, conf.BandwidthLimit/conf.ChunkSize, shares_chan, bw_limited_chunks)
	udpsender.CreateSender(ctx, conf.ReceiverIP, conf.ReceiverPort, bw_limited_chunks, 20)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	cancel() // Gracefully shutdown and stop all goroutines
}
