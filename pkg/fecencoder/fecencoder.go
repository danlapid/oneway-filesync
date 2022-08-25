package fecencoder

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/structs"

	"github.com/klauspost/reedsolomon"
	"github.com/sirupsen/logrus"
)

type FecEncoder struct {
	chunksize int
	required  int
	total     int
	input     chan *structs.Chunk
	output    chan *structs.Chunk
}

// FEC routine:
// For each part of the <total> parts we make a realchunksize/<required> share
// These are encoding using reed solomon FEC
// Then we send each share seperately
// At the end they are combined and concatenated to form the file.
func Worker(ctx context.Context, conf *FecEncoder) {
	fec, err := reedsolomon.New(conf.required, conf.total-conf.required)
	if err != nil {
		logrus.Errorf("Error creating fec object: %v", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-conf.input:
			padding := (conf.required - (len(chunk.Data) % conf.required)) % conf.required
			chunk.Data = append(chunk.Data, make([]byte, padding)...)

			// Split the data into shares
			shares, err := fec.Split(chunk.Data)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Path": chunk.Path,
					"Hash": fmt.Sprintf("%x", chunk.Hash),
				}).Errorf("Error splitting chunk: %v", err)
				continue
			}

			// Encode the parity set
			err = fec.Encode(shares)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Path": chunk.Path,
					"Hash": fmt.Sprintf("%x", chunk.Hash),
				}).Errorf("Error FEC encoding chunk: %v", err)
				continue
			}

			for i, sharedata := range shares {
				chunk := structs.Chunk{
					Path:        chunk.Path,
					Hash:        chunk.Hash,
					DataOffset:  chunk.DataOffset,
					DataPadding: uint32(padding),
					ShareIndex:  uint32(i),
					Data:        sharedata,
				}
				conf.output <- &chunk
			}
		}
	}
}

func CreateFecEncoder(ctx context.Context, chunksize int, required int, total int, input chan *structs.Chunk, output chan *structs.Chunk, workercount int) {
	conf := FecEncoder{
		chunksize: chunksize,
		required:  required,
		total:     total,
		input:     input,
		output:    output,
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, &conf)
	}
}
