package bandwidthlimiter_test

import (
	"context"
	"oneway-filesync/pkg/bandwidthlimiter"
	"oneway-filesync/pkg/structs"
	"runtime"
	"testing"
	"time"
)

func TestCreateBandwidthLimiter(t *testing.T) {
	type args struct {
		chunk_count   int
		chunk_size    int
		bytes_per_sec int
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "test1", args: args{chunk_count: 100, chunk_size: 8000, bytes_per_sec: 240000}},
		{name: "test2", args: args{chunk_count: 300, chunk_size: 8000, bytes_per_sec: 240000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := (float64(tt.args.chunk_size) / float64(tt.args.bytes_per_sec)) * float64(tt.args.chunk_count)
			ch_in := make(chan *structs.Chunk, tt.args.chunk_count)
			ch_out := make(chan *structs.Chunk, tt.args.chunk_count)

			chunk := structs.Chunk{Data: make([]byte, tt.args.chunk_size)}
			for i := 0; i < tt.args.chunk_count; i++ {
				ch_in <- &chunk
			}

			ctx, cancel := context.WithCancel(context.Background())
			start := time.Now()
			bandwidthlimiter.CreateBandwidthLimiter(ctx, tt.args.bytes_per_sec, tt.args.chunk_size, ch_in, ch_out, runtime.GOMAXPROCS(0)*2)

			for i := 0; i < tt.args.chunk_count; i++ {
				<-ch_out
			}
			timepast := time.Since(start)

			if timepast > time.Duration(expected*1.2)*time.Second || timepast < time.Duration(expected*0.8)*time.Second {
				t.Fatalf("Bandwidthlimiter took %f seconds instead of %f", timepast.Seconds(), expected)
			}
			cancel()
		})
	}
}
