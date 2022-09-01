package filereader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/structs"
	"os"
	"path/filepath"

	"github.com/alexmullins/zip"
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

func reader(f *os.File, w io.Writer) error {
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("error copying file contents: %v", err)
	}
	return nil
}

func encryptedreader(f *os.File, w io.Writer) error {
	ziparchive := zip.NewWriter(w)
	zipfile, err := ziparchive.Encrypt(filepath.Base(f.Name()), `filesync`)
	if err != nil {
		return fmt.Errorf("error creating file in zip: %v", err)
	}

	if _, err = io.Copy(zipfile, f); err != nil {
		return fmt.Errorf("error copying file contents: %v", err)
	}

	if err = ziparchive.Close(); err != nil {
		return fmt.Errorf("error closing zip file: %v", err)
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
			realchunksize := conf.chunksize - structs.ChunkOverhead(file.Path)
			realchunksize *= conf.required // FEC chunk size is BuffferSize/Required

			l := logrus.WithFields(logrus.Fields{
				"Path": file.Path,
				"Hash": fmt.Sprintf("%x", file.Hash),
			})
			l.Infof("Started sending file")

			success := true
			w := chunkWriter{
				chunksize: 8192,
				sendchunk: func(data []byte, offset int64) {
					chunk := structs.Chunk{
						Path:       file.Path,
						DataOffset: offset,
						Data:       data,
					}
					copy(chunk.Hash[:], file.Hash)
					conf.output <- &chunk
				},
			}

			f, err := os.Open(file.Path)
			if err != nil {
				l.Errorf("error opening file: %v", err)
				continue
			}
			defer f.Close()

			if conf.encrypted {
				err = encryptedreader(f, &w)
			} else {
				err = reader(f, &w)
			}

			if err != nil {
				success = false
				l.Error(err)
				continue
			}

			err = w.Close()
			if err != nil {
				success = false
				l.Error(err)
				continue
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
