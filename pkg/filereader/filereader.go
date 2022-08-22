// Reads file paths to be sent from the queue
package filereader

import (
	"bufio"
	"context"
	"io"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/structs"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/vivint/infectious"
)

type FileReader struct {
	chunksize int
	required  int
	total     int
	input     chan database.File
	output    chan []byte
}

func Worker(ctx context.Context, conf FileReader) {
	fec, err := infectious.NewFEC(conf.required, conf.total)
	if err != nil {
		logrus.Errorf("Error creating fec object: %v", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case file := <-conf.input:

			realchunksize := conf.chunksize - structs.ChunkOverhead(file.Path)
			realchunksize *= conf.required // FEC chunk size is BuffferSize/Required

			logrus.WithFields(logrus.Fields{
				"Path": file.Path,
				"Hash": file.Hash,
			}).Infof("Started sending file")

			f, err := os.Open(file.Path)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Path": file.Path,
					"Hash": file.Hash,
				}).Errorf("Error opening file: %v", err)
				continue
			}
			defer f.Close()

			var i uint32 = 0
			success := false
			r := bufio.NewReader(f)
			for {
				data := make([]byte, realchunksize)
				n, err := r.Read(data)
				if err != nil {
					if n == 0 && err == io.EOF {
						success = true
						break
					}
					logrus.WithFields(logrus.Fields{
						"Path": file.Path,
						"Hash": file.Hash,
					}).Errorf("Error reading file: %v", err)
					break
				}

				// FEC routine:
				// For each part of the <total> parts we make a realchunksize/<required> share
				// These are encoding using reed solomon FEC
				// Then we send each share seperately
				// At the end they are combined and concatenated to form the file.
				// TODO: export this part to another pipe
				for j := 0; j < conf.total; j++ {
					sharedata := make([]byte, realchunksize/conf.required)
					err = fec.EncodeSingle(data[:], sharedata, j)
					if err != nil {
						logrus.WithFields(logrus.Fields{
							"Path": file.Path,
							"Hash": file.Hash,
						}).Errorf("Error FECing chunk: %v", err)
						break
					}

					chunk := structs.Chunk{
						Path:      file.Path,
						DataIndex: i,
						Data:      sharedata,
					}
					copy(chunk.Hash[:], file.Hash)

					chunksharedata, err := chunk.Encode()
					if err != nil {
						logrus.WithFields(logrus.Fields{
							"Path": file.Path,
							"Hash": file.Hash,
						}).Errorf("Error encoding chunk: %v", err)
						break
					}
					conf.output <- chunksharedata
				}

				i++
			}
			file.Finished = true
			file.Success = success
			logrus.WithFields(logrus.Fields{
				"Path": file.Path,
				"Hash": file.Hash,
			}).Infof("File finished sending, Success=%t", success)
			err = database.UpdateFileInDatabase(file)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Path": file.Path,
					"Hash": file.Hash,
				}).Errorf("Error updating Finished in database %v", err)
			}
		}
	}
}

func CreateFileReader(ctx context.Context, chunksize int, required int, total int, input chan database.File, output chan []byte, workercount int) {
	conf := FileReader{
		chunksize: chunksize,
		required:  required,
		total:     total,
		input:     input,
		output:    output,
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, conf)
	}
}
