package receiver

import (
	"context"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/fecdecoder"
	"oneway-filesync/pkg/filecloser"
	"oneway-filesync/pkg/filewriter"
	"oneway-filesync/pkg/shareassembler"
	"oneway-filesync/pkg/structs"
	"oneway-filesync/pkg/udpreceiver"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Receiver(ctx context.Context, db *gorm.DB, conf config.Config) {
	tmpdir := filepath.Join(conf.OutDir, "tempfiles")
	err := os.MkdirAll(tmpdir, os.ModePerm)
	if err != nil {
		logrus.Errorf("Failed creating tempdir with err %v\n", err)
		return
	}

	shares_chan := make(chan *structs.Chunk, 100)
	sharelist_chan := make(chan []*structs.Chunk, 100)
	chunks_chan := make(chan *structs.Chunk, 100)
	finishedfiles_chan := make(chan *structs.OpenTempFile, 5)

	udpreceiver.CreateUdpReceiver(ctx, conf.ReceiverIP, conf.ReceiverPort, conf.ChunkSize, shares_chan, 20)
	shareassembler.CreateShareAssembler(ctx, conf.ChunkFecRequired, conf.ChunkFecTotal, shares_chan, sharelist_chan, 20)
	fecdecoder.CreateFecDecoder(ctx, conf.ChunkFecRequired, conf.ChunkFecTotal, sharelist_chan, chunks_chan, 20)
	filewriter.CreateFileWriter(ctx, tmpdir, chunks_chan, finishedfiles_chan, 20)
	filecloser.CreateFileCloser(ctx, db, conf.OutDir, finishedfiles_chan, 5)
}
