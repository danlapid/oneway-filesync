package sender

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
	"runtime"

	"gorm.io/gorm"
)

func Sender(ctx context.Context, db *gorm.DB, conf config.Config) {
	maxprocs := runtime.GOMAXPROCS(0) * 2
	queue_chan := make(chan database.File, 10)
	chunks_chan := make(chan *structs.Chunk, 100)
	shares_chan := make(chan *structs.Chunk, 100)
	bw_limited_chunks := make(chan *structs.Chunk, 5) // Small buffer to reduce burst

	queuereader.CreateQueueReader(ctx, db, queue_chan)
	filereader.CreateFileReader(ctx, db, conf.ChunkSize, conf.ChunkFecRequired, queue_chan, chunks_chan, maxprocs)
	fecencoder.CreateFecEncoder(ctx, conf.ChunkSize, conf.ChunkFecRequired, conf.ChunkFecTotal, chunks_chan, shares_chan, maxprocs)
	bandwidthlimiter.CreateBandwidthLimiter(ctx, conf.BandwidthLimit/conf.ChunkSize, shares_chan, bw_limited_chunks)
	udpsender.CreateSender(ctx, conf.ReceiverIP, conf.ReceiverPort, bw_limited_chunks, maxprocs)
}
