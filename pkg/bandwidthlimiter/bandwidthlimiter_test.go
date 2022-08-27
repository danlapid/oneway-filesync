package bandwidthlimiter_test

import (
	"context"
	"oneway-filesync/pkg/bandwidthlimiter"
	"oneway-filesync/pkg/structs"
	"testing"
	"time"
)

func TestCreateBandwidthLimiter(t *testing.T) {
	type args struct {
		chunks         int
		chunks_per_sec int
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "test1", args: args{chunks: 100, chunks_per_sec: 10}},
		{name: "test2", args: args{chunks: 100000, chunks_per_sec: 10000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := float64(tt.args.chunks) / float64(tt.args.chunks_per_sec)
			ch_in := make(chan *structs.Chunk, tt.args.chunks)
			ch_out := make(chan *structs.Chunk, tt.args.chunks)
			for i := 0; i < tt.args.chunks; i++ {
				ch_in <- &structs.Chunk{}
			}
			ctx, cancel := context.WithCancel(context.Background())
			start := time.Now()
			bandwidthlimiter.CreateBandwidthLimiter(ctx, tt.args.chunks_per_sec, ch_in, ch_out)
			for i := 0; i < tt.args.chunks; i++ {
				<-ch_out
			}
			timepast := time.Since(start)
			if timepast > time.Duration(expected+1)*time.Second || timepast < time.Duration(expected-1)*time.Second {
				t.Fatalf("Bandwidthlimiter took %f seconds instead of %f", timepast.Seconds(), expected)
			}
			cancel()
		})
	}
}
