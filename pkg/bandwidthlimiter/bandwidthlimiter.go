package bandwidthlimiter

import (
	"context"

	"go.uber.org/ratelimit"
)

type BandwidthLimiter struct {
	rl     ratelimit.Limiter
	input  chan []byte
	output chan []byte
}

func Worker(ctx context.Context, conf BandwidthLimiter) {
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

func CreateBandwidthLimiter(ctx context.Context, chunks_per_sec int, input chan []byte, output chan []byte) {
	conf := BandwidthLimiter{
		rl:     ratelimit.New(chunks_per_sec),
		input:  input,
		output: output,
	}
	go Worker(ctx, conf)
}
