package bandwidthlimiter

import (
	"context"
	"oneway-filesync/pkg/structs"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type bandwidthLimiterConfig struct {
	rl     *rate.Limiter
	input  chan *structs.Chunk
	output chan *structs.Chunk
}

func worker(ctx context.Context, conf *bandwidthLimiterConfig) {
	for {
		select {
		case <-ctx.Done():
			return
		case buf := <-conf.input:
			if err := conf.rl.WaitN(ctx, len(buf.Data)); err != nil {
				logrus.Error(err)
			} else {
				conf.output <- buf
			}
		}
	}
}

func CreateBandwidthLimiter(ctx context.Context, bandwidth int, chunksize int, input chan *structs.Chunk, output chan *structs.Chunk, workercount int) {
	conf := bandwidthLimiterConfig{
		rl:     rate.NewLimiter(rate.Limit(bandwidth), chunksize),
		input:  input,
		output: output,
	}
	for i := 0; i < workercount; i++ {
		go worker(ctx, &conf)
	}
}
