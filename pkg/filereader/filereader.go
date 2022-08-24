package filereader

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/structs"
	"os"

	"github.com/sirupsen/logrus"
)

type FileReader struct {
	chunksize int
	required  int
	input     chan database.File
	output    chan structs.Chunk
}

func Worker(ctx context.Context, conf *FileReader) {
	db, err := database.OpenDatabase()
	if err != nil {
		logrus.Errorf("Error connecting to the database: %v", err)
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
				"Hash": fmt.Sprintf("%x", file.Hash),
			}).Infof("Started sending file")

			f, err := os.Open(file.Path)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Path": file.Path,
					"Hash": fmt.Sprintf("%x", file.Hash),
				}).Errorf("Error opening file: %v", err)
				continue
			}
			defer f.Close()

			var offset int64 = 0
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
						"Hash": fmt.Sprintf("%x", file.Hash),
					}).Errorf("Error reading file: %v", err)
					break
				}

				chunk := structs.Chunk{
					Path:       file.Path,
					DataOffset: offset,
					Data:       data[:n],
				}
				copy(chunk.Hash[:], file.Hash)
				conf.output <- chunk
				offset += int64(n)
			}
			file.Finished = true
			file.Success = success
			logrus.WithFields(logrus.Fields{
				"Path": file.Path,
				"Hash": fmt.Sprintf("%x", file.Hash),
			}).Infof("File finished sending, Success=%t", success)
			err = db.Save(&file).Error
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Path": file.Path,
					"Hash": fmt.Sprintf("%x", file.Hash),
				}).Errorf("Error updating Finished in database %v", err)
			}
		}
	}
}

func CreateFileReader(ctx context.Context, chunksize int, required int, input chan database.File, output chan structs.Chunk, workercount int) {
	conf := FileReader{
		chunksize: chunksize,
		required:  required,
		input:     input,
		output:    output,
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, &conf)
	}
}
