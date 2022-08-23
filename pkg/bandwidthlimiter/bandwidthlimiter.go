package bandwidthlimiter

import (
	"context"
	"oneway-filesync/pkg/structs"

	"go.uber.org/ratelimit"
)

type BandwidthLimiter struct {
	rl     ratelimit.Limiter
	input  chan structs.Chunk
	output chan structs.Chunk
}

func Worker(ctx context.Context, conf *BandwidthLimiter) {
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

func CreateBandwidthLimiter(ctx context.Context, chunks_per_sec int, input chan structs.Chunk, output chan structs.Chunk) {
	conf := BandwidthLimiter{
		rl:     ratelimit.New(chunks_per_sec),
		input:  input,
		output: output,
	}
	go Worker(ctx, &conf)
}
