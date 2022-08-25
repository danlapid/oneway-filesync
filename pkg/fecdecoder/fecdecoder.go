package fecdecoder

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/structs"
	"time"

	"github.com/klauspost/reedsolomon"
	"github.com/sirupsen/logrus"
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
	input    chan []*structs.Chunk
	output   chan *structs.Chunk
}

func Worker(ctx context.Context, conf *FecDecoder) {
	fec, err := reedsolomon.New(conf.required, conf.total-conf.required)
	if err != nil {
		logrus.Errorf("Error creating fec object: %v", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case chunks := <-conf.input:
			shares := make([][]byte, conf.total)
			for _, chunk := range chunks {
				shares[chunk.ShareIndex] = chunk.Data
			}
			err := fec.ReconstructData(shares)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Path": chunks[0].Path,
					"Hash": fmt.Sprintf("%x", chunks[0].Hash),
				}).Errorf("Error FEC decoding shares: %v", err)
				continue
			}

			data := make([]byte, len(shares[0])*conf.required)
			for i, shard := range shares[:conf.required] {
				copy(data[i*len(shares[0]):], shard)
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

func CreateFecDecoder(ctx context.Context, required int, total int, input chan []*structs.Chunk, output chan *structs.Chunk, workercount int) {
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
