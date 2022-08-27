package filewriter

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/structs"
	"oneway-filesync/pkg/utils"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func pathReplace(path string) string {
	newpath := path
	newpath = strings.ReplaceAll(newpath, "/", "_")
	newpath = strings.ReplaceAll(newpath, "\\", "_")
	newpath = strings.ReplaceAll(newpath, ":", "_")
	return newpath
}

type fileWriterConfig struct {
	tempdir string
	input   chan *structs.Chunk
	output  chan *structs.OpenTempFile
	cache   utils.RWMutexMap[string, *structs.OpenTempFile]
}

// The manager acts as a "closer"
// Since we can never really be sure all the chunks arrive
// But 30 seconds after no more chunks arrive we can be rather certain
// no more chunks will arrive
func manager(ctx context.Context, conf *fileWriterConfig) {
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conf.cache.Range(func(tempfilepath string, value *structs.OpenTempFile) bool {
				if time.Since(value.LastUpdated).Seconds() > 30 {
					conf.cache.Delete(tempfilepath)
					conf.output <- value
				}
				return true
			})
		}
	}
}

func worker(ctx context.Context, conf *fileWriterConfig) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-conf.input:
			tempfilepath := filepath.Join(conf.tempdir, fmt.Sprintf("%s___%x.tmp", pathReplace(chunk.Path), chunk.Hash))
			tempfile, err := os.OpenFile(tempfilepath, os.O_RDWR|os.O_CREATE, 0600)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": tempfilepath,
					"Path":     chunk.Path,
					"Hash":     fmt.Sprintf("%x", chunk.Hash),
				}).Errorf("Error creating tempfile for chunk: %v", err)
				continue
			}

			_, err = tempfile.WriteAt(chunk.Data, chunk.DataOffset)
			tempfile.Close() // Not using defer because of overhead concerns
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": tempfilepath,
					"Path":     chunk.Path,
					"Hash":     fmt.Sprintf("%x", chunk.Hash),
				}).Errorf("Error writing to tempfile: %v", err)
				continue
			}

			conf.cache.Store(tempfilepath, &structs.OpenTempFile{
				TempFile:    tempfilepath,
				Path:        chunk.Path,
				Hash:        chunk.Hash,
				LastUpdated: time.Now(),
			})
		}
	}
}

func CreateFileWriter(ctx context.Context, tempdir string, input chan *structs.Chunk, output chan *structs.OpenTempFile, workercount int) {
	conf := fileWriterConfig{
		tempdir: tempdir,
		input:   input,
		output:  output,
		cache:   utils.RWMutexMap[string, *structs.OpenTempFile]{},
	}
	for i := 0; i < workercount; i++ {
		go worker(ctx, &conf)
	}
	go manager(ctx, &conf)
}
