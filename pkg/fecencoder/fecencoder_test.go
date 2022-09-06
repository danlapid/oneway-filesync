package fecencoder

import (
	"bytes"
	"context"
	"oneway-filesync/pkg/structs"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func Test_worker(t *testing.T) {
	type args struct {
		required int
		total    int
		input    *structs.Chunk
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr string
	}{
		{"test-works", args{2, 4, &structs.Chunk{Data: make([]byte, 400)}}, false, ""},
		{"test-shortdata1", args{2, 4, &structs.Chunk{Data: make([]byte, 0)}}, true, "Error splitting chunk: not enough data to fill the number of requested shards"},
		{"test-invalid-fec1", args{2, 1, &structs.Chunk{}}, true, "Error creating fec object: cannot create Encoder with less than one data shard or less than zero parity shards"},
		{"test-invalid-fec2", args{0, 1, &structs.Chunk{}}, true, "Error creating fec object: cannot create Encoder with less than one data shard or less than zero parity shards"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var memLog bytes.Buffer
			logrus.SetOutput(&memLog)

			input := make(chan *structs.Chunk, 5)
			output := make(chan *structs.Chunk, 5)

			input <- tt.args.input

			conf := fecEncoderConfig{tt.args.required, tt.args.total, input, output}
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(2 * time.Second)
				cancel()
			}()
			// ch <- tt.args.file
			worker(ctx, &conf)

			if tt.wantErr {
				if !strings.Contains(memLog.String(), tt.expectedErr) {
					t.Fatalf("Expected not in log, '%v' not in '%v'", tt.expectedErr, memLog.String())
				}
			} else {
				for i := 0; i < conf.total; i++ {
					<-output
				}
			}
		})
	}
}
