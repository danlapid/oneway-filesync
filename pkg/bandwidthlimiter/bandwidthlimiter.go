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

func CreateBandwidthLimiter(ctx context.Context, bytes_per_sec int, input chan *structs.Chunk, output chan *structs.Chunk) {
	conf := bandwidthLimiterConfig{
		rl:     rate.NewLimiter(rate.Limit(bytes_per_sec), bytes_per_sec),
		input:  input,
		output: output,
	}
	go worker(ctx, &conf)
}
