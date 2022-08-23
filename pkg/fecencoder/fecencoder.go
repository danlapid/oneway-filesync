package fecencoder

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/structs"

	"github.com/sirupsen/logrus"
	"github.com/vivint/infectious"
)

type FecEncoder struct {
	chunksize int
	required  int
	total     int
	input     chan structs.Chunk
	output    chan structs.Chunk
}

func Worker(ctx context.Context, conf *FecEncoder) {
	fec, err := infectious.NewFEC(conf.required, conf.total)
	if err != nil {
		logrus.Errorf("Error creating fec object: %v", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-conf.input:

			sharesize := conf.chunksize - structs.ChunkOverhead(chunk.Path)

			// FEC routine:
			// For each part of the <total> parts we make a realchunksize/<required> share
			// These are encoding using reed solomon FEC
			// Then we send each share seperately
			// At the end they are combined and concatenated to form the file.
			for i := 0; i < conf.total; i++ {
				sharedata := make([]byte, sharesize)
				err = fec.EncodeSingle(chunk.Data[:], sharedata, i)
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"Path": chunk.Path,
						"Hash": fmt.Sprintf("%x", chunk.Hash),
					}).Errorf("Error FEC encoding chunk: %v", err)
					break
				}

				chunk := structs.Chunk{
					Path:       chunk.Path,
					Size:       chunk.Size,
					Hash:       chunk.Hash,
					DataOffset: chunk.DataOffset,
					ShareIndex: uint32(i),
					Data:       sharedata,
				}

				conf.output <- chunk
			}

		}
	}
}

func CreateFecEncoder(ctx context.Context, chunksize int, required int, total int, input chan structs.Chunk, output chan structs.Chunk, workercount int) {
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
