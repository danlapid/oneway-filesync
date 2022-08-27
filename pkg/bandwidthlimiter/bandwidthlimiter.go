package bandwidthlimiter

import (
	"context"
	"oneway-filesync/pkg/structs"

	"go.uber.org/ratelimit"
)

type bandwidthLimiterConfig struct {
	rl     ratelimit.Limiter
	input  chan *structs.Chunk
	output chan *structs.Chunk
}

func worker(ctx context.Context, conf *bandwidthLimiterConfig) {
	for {
		select {
		case <-ctx.Done():
			return
		case buf := <-conf.input:
			conf.rl.Take()
			conf.output <- buf
		}
	}
}

func CreateBandwidthLimiter(ctx context.Context, chunks_per_sec int, input chan *structs.Chunk, output chan *structs.Chunk) {
	conf := bandwidthLimiterConfig{
		rl:     ratelimit.New(chunks_per_sec),
		input:  input,
		output: output,
	}
	go worker(ctx, &conf)
}
