package filewriter

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/structs"
	"oneway-filesync/pkg/utils.go"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type FileWriter struct {
	tempdir string
	input   chan structs.Chunk
	output  chan structs.OpenTempFile
	cache   utils.RWMutexMap[string, *structs.OpenTempFile]
}

// The manager acts as a "closer"
// Since we can never really be sure all the chunks arrive
// But 30 seconds after no more chunks arrive we can be rather certain
// no more chunks will arrive
func Manager(ctx context.Context, conf *FileWriter) {
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conf.cache.Range(func(tempfilepath string, value *structs.OpenTempFile) bool {
				if time.Since(value.LastUpdated).Seconds() > 30 {
					conf.cache.Delete(tempfilepath)
					conf.output <- *value
					logrus.Infof("Sent file")
				}
				return true
			})
		}
	}
}

func Worker(ctx context.Context, conf *FileWriter) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-conf.input:
			tempfile, err := os.CreateTemp(conf.tempdir, fmt.Sprintf("%s___%x___*.tmp", strings.ReplaceAll(chunk.Path, "/", "_"), chunk.Hash))
			tempfilepath := tempfile.Name()
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": tempfilepath,
					"Path":     chunk.Path,
					"Hash":     fmt.Sprintf("%x", chunk.Hash),
				}).Errorf("Error creating tempfile for chunk: %v", err)
				continue
			}

			_, err = tempfile.WriteAt(chunk.Data, chunk.DataOffset)
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
				Size:        chunk.Size,
				Hash:        chunk.Hash,
				LastUpdated: time.Now(),
			})
		}
	}
}

func CreateFileWriter(ctx context.Context, tempdir string, input chan structs.Chunk, output chan structs.OpenTempFile, workercount int) {
	conf := FileWriter{
		tempdir: tempdir,
		input:   input,
		output:  output,
		cache:   utils.RWMutexMap[string, *structs.OpenTempFile]{},
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, &conf)
	}
	go Manager(ctx, &conf)
}
