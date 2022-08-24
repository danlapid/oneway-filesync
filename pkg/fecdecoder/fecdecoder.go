package fecdecoder

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/structs"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vivint/infectious"
)

// Cache docs:
// For every (FileHash,FileDataOffset) we save a cache of shares
// Since we need at least <required> shares to create the original data we have to cache them somewhere
// After we get <required> shares we can pull them and create the data but then up to (<total>-<required>) will continue coming in
// The LastUpdated is a field which we can time out based upon and
type CacheKey struct {
	Hash       [structs.HASHSIZE]byte
	DataOffset int64
}
type CacheValue struct {
	Shares      chan *structs.Chunk
	LastUpdated time.Time
}

type FecDecoder struct {
	required int
	total    int
	input    chan []structs.Chunk
	output   chan *structs.Chunk
}

// m := make(map[string]int)
func Worker(ctx context.Context, conf *FecDecoder) {
	fec, err := infectious.NewFEC(conf.required, conf.total)
	if err != nil {
		logrus.Errorf("Error creating fec object: %v", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case chunks := <-conf.input:
			shares := make([]infectious.Share, len(chunks))
			for i, chunk := range chunks {
				shares[i] = infectious.Share{
					Number: int(chunk.ShareIndex),
					Data:   chunk.Data,
				}
			}
			data, err := fec.Decode(nil, shares)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Path": chunks[0].Path,
					"Hash": fmt.Sprintf("%x", chunks[0].Hash),
				}).Errorf("Error FEC decoding shares: %v", err)
				continue
			}

			conf.output <- &structs.Chunk{
				Path:       chunks[0].Path,
				Hash:       chunks[0].Hash,
				DataOffset: chunks[0].DataOffset,
				Data:       data[:len(data)-int(chunks[0].DataPadding)],
			}
		}
	}
}

func CreateFecDecoder(ctx context.Context, required int, total int, input chan []structs.Chunk, output chan *structs.Chunk, workercount int) {
	conf := FecDecoder{
		required: required,
		total:    total,
		input:    input,
		output:   output,
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, &conf)
	}
}
