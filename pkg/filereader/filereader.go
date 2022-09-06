package filereader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/structs"
	"oneway-filesync/pkg/zip"
	"os"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type chunkWriter struct {
	buf       bytes.Buffer
	chunksize int
	offset    int64
	sendchunk func(data []byte, offset int64)
}

func (w *chunkWriter) dumpChunk() {
	b := make([]byte, w.chunksize)
	n, _ := w.buf.Read(b) // err means EOF
	if n > 0 {
		w.sendchunk(b[:n], w.offset)
		w.offset += int64(n)
	}
}

func (w *chunkWriter) Write(p []byte) (int, error) {
	_, _ = w.buf.Write(p) // bytes.Buffer.Write never returns error
	if w.buf.Len() > w.chunksize {
		w.dumpChunk()
	}
	return len(p), nil
}

func (w *chunkWriter) Close() {
	for {
		if w.buf.Len() == 0 {
			break
		}
		w.dumpChunk()
	}
}

func sendfile(file *database.File, conf *fileReaderConfig) error {
	realchunksize := conf.chunksize - structs.ChunkOverhead(file.Path)
	realchunksize *= conf.required // FEC chunk size is BuffferSize/Required

	w := chunkWriter{
		chunksize: realchunksize,
		sendchunk: func(data []byte, offset int64) {
			chunk := structs.Chunk{
				Path:       file.Path,
				Encrypted:  file.Encrypted,
				DataOffset: offset,
				Data:       data,
			}
			copy(chunk.Hash[:], file.Hash)
			conf.output <- &chunk
		},
	}

	f, err := os.Open(file.Path)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close()

	if file.Encrypted {
		err = zip.ZipFile(&w, f)
	} else {
		_, err = io.Copy(&w, f)
	}
	if err != nil {
		return err
	}

	w.Close()
	return nil
}

type fileReaderConfig struct {
	db        *gorm.DB
	chunksize int
	required  int
	input     chan database.File
	output    chan *structs.Chunk
}

func worker(ctx context.Context, conf *fileReaderConfig) {
	for {
		select {
		case <-ctx.Done():
			return
		case file := <-conf.input:
			l := logrus.WithFields(logrus.Fields{
				"Path": file.Path,
				"Hash": fmt.Sprintf("%x", file.Hash),
			})
			l.Infof("Started sending file")

			err := sendfile(&file, conf)
			if err != nil {
				file.Success = false
				l.Errorf("File sending failed with err: %v", err)
			} else {
				file.Success = true
				l.Infof("File successfully finished sending")

			}

			file.Finished = true
			err = conf.db.Save(&file).Error
			if err != nil {
				l.Errorf("Error updating Finished in database %v", err)
			}
		}
	}
}

func CreateFileReader(ctx context.Context, db *gorm.DB, chunksize int, required int, input chan database.File, output chan *structs.Chunk, workercount int) {
	conf := fileReaderConfig{
		db:        db,
		chunksize: chunksize,
		required:  required,
		input:     input,
		output:    output,
	}
	for i := 0; i < workercount; i++ {
		go worker(ctx, &conf)
	}
}
