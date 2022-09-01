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

func (w *chunkWriter) dumpChunk() error {
	b := make([]byte, w.chunksize)
	n, err := w.buf.Read(b)
	if err != nil {
		return err
	}
	w.sendchunk(b[:n], w.offset)
	w.offset += int64(n)
	return nil
}

func (w *chunkWriter) Write(p []byte) (int, error) {
	_, err := w.buf.Write(p)
	if err != nil {
		return 0, err
	}
	if w.buf.Len() > w.chunksize {
		err := w.dumpChunk()
		if err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (w *chunkWriter) Close() error {
	for {
		if w.buf.Len() == 0 {
			break
		}
		err := w.dumpChunk()
		if err != nil {
			return err
		}
	}
	return nil
}

func sendfile(file *database.File, conf *fileReaderConfig) error {
	filepath := file.Path
	if conf.encrypted {
		filepath += ".zip"
	}
	realchunksize := conf.chunksize - structs.ChunkOverhead(filepath)
	realchunksize *= conf.required // FEC chunk size is BuffferSize/Required

	w := chunkWriter{
		chunksize: realchunksize,
		sendchunk: func(data []byte, offset int64) {
			chunk := structs.Chunk{
				Path:       filepath,
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

	if conf.encrypted {
		err = zip.ZipFile(&w, f)
	} else {
		_, err = io.Copy(&w, f)
	}
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}
	return nil
}

type fileReaderConfig struct {
	db        *gorm.DB
	chunksize int
	encrypted bool
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

			success := true
			err := sendfile(&file, conf)
			if err != nil {
				success = false
				l.Error(err)
			}

			file.Finished = true
			file.Success = success
			l.Infof("File finished sending, Success=%t", success)
			err = conf.db.Save(&file).Error
			if err != nil {
				l.Errorf("Error updating Finished in database %v", err)
			}
		}
	}
}

func CreateFileReader(ctx context.Context, db *gorm.DB, chunksize int, encrypted bool, required int, input chan database.File, output chan *structs.Chunk, workercount int) {
	conf := fileReaderConfig{
		db:        db,
		chunksize: chunksize,
		encrypted: encrypted,
		required:  required,
		input:     input,
		output:    output,
	}
	for i := 0; i < workercount; i++ {
		go worker(ctx, &conf)
	}
}
