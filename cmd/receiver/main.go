package main

import (
	"context"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/fecdecoder"
	"oneway-filesync/pkg/filecloser"
	"oneway-filesync/pkg/filewriter"
	"oneway-filesync/pkg/shareassembler"
	"oneway-filesync/pkg/structs"
	"oneway-filesync/pkg/udpreceiver"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	conf, err := config.GetConfig("config.toml")
	if err != nil {
		logrus.Errorf("Failed reading config with err %v\n", err)
		return

	}

	err = database.ConfigureDatabase()
	if err != nil {
		logrus.Errorf("Failed setting up db with err %v\n", err)
		return
	}
	tmpdir := filepath.Join(conf.OutDir, "tempfiles")
	err = os.MkdirAll(tmpdir, os.ModePerm)
	if err != nil {
		logrus.Errorf("Failed creating tempdir with err %v\n", err)
		return
	}

	shares_chan := make(chan structs.Chunk, 100)
	sharelist_chan := make(chan []structs.Chunk, 100)
	chunks_chan := make(chan structs.Chunk, 100)
	finishedfiles_chan := make(chan structs.OpenTempFile, 10)

	ctx, cancel := context.WithCancel(context.Background()) // Create a cancelable context and pass it to all goroutines, allows us to gracefully shut down the program

	udpreceiver.CreateReceiver(ctx, conf.ReceiverIP, conf.ReceiverPort, conf.ChunkSize, shares_chan, 20)
	shareassembler.CreateShareAssembler(ctx, conf.ChunkFecRequired, conf.ChunkFecTotal, shares_chan, sharelist_chan, 20)
	fecdecoder.CreateFecDecoder(ctx, conf.ChunkFecRequired, conf.ChunkFecTotal, sharelist_chan, chunks_chan, 20)
	filewriter.CreateFileWriter(ctx, tmpdir, chunks_chan, finishedfiles_chan, 20)
	filecloser.CreateFileCloser(ctx, conf.OutDir, finishedfiles_chan, 5)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	cancel() // Gracefully shutdown and stop all goroutines
}
