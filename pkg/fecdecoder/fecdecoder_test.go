package fecdecoder

import (
	"bytes"
	"context"
	"oneway-filesync/pkg/structs"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/reedsolomon"
	"github.com/sirupsen/logrus"
)

func createChunks(t *testing.T, required int, total int) []*structs.Chunk {
	fec, err := reedsolomon.New(required, total-required)
	if err != nil {
		t.Fatal(err)
	}
	shares, err := fec.Split(make([]byte, 400))
	if err != nil {
		t.Fatal(err)
	}

	// Encode the parity set
	err = fec.Encode(shares)
	if err != nil {
		t.Fatal(err)
	}
	chunks := make([]*structs.Chunk, total)
	for i, sharedata := range shares {
		chunks[i] = &structs.Chunk{
			ShareIndex: uint32(i),
			Data:       sharedata,
		}
	}
	return chunks

}

func Test_worker(t *testing.T) {
	type args struct {
		required int
		total    int
		input    []*structs.Chunk
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr string
	}{
		{"test-works", args{2, 4, createChunks(t, 2, 4)}, false, ""},
		{"test-too-few-shards", args{4, 8, createChunks(t, 4, 8)[:3]}, true, "Error FEC decoding shares: too few shards given"},
		{"test-invalid-fec1", args{2, 1, make([]*structs.Chunk, 4)}, true, "Error creating fec object: cannot create Encoder with less than one data shard or less than zero parity shards"},
		{"test-invalid-fec2", args{0, 1, make([]*structs.Chunk, 4)}, true, "Error creating fec object: cannot create Encoder with less than one data shard or less than zero parity shards"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var memLog bytes.Buffer
			logrus.SetOutput(&memLog)

			input := make(chan []*structs.Chunk, 5)
			output := make(chan *structs.Chunk, 5)

			input <- tt.args.input

			conf := fecDecoderConfig{tt.args.required, tt.args.total, input, output}
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
				<-output
			}

		})
	}
}
