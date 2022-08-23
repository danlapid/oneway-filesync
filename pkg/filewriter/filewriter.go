package filewriter

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/structs"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type FileWriter struct {
	tempdir string
	input   chan structs.Chunk
	output  chan structs.OpenTempFile
	cache   sync.Map
}

// The manager acts as a "closer"
// Since we can never really be sure all the chunks arrive
// But 30 seconds after no more chunks arrive we can be rather certain
// no more chunks will arrive
func Manager(ctx context.Context, conf FileWriter) {
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conf.cache.Range(func(k, v interface{}) bool {
				tempfilepath, _ := k.(string)
				value, _ := v.(structs.OpenTempFile)
				if time.Since(value.LastUpdated) > 30*time.Second {
					conf.cache.Delete(tempfilepath)
				}
				conf.output <- value
				return true
			})
		}
	}
}

func Worker(ctx context.Context, conf FileWriter) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-conf.input:
			tempfile, err := os.CreateTemp(conf.tempdir, fmt.Sprintf("%s___%x___*.tmp", strings.ReplaceAll(chunk.Path, "/", "_"), chunk.Hash))
			tempfilepath := conf.tempdir + tempfile.Name()
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": tempfilepath,
					"Path":     chunk.Path,
					"Hash":     chunk.Hash,
				}).Errorf("Error creating tempfile for chunk: %v", err)
				continue
			}

			_, err = tempfile.WriteAt(chunk.Data, chunk.DataOffset)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": tempfilepath,
					"Path":     chunk.Path,
					"Hash":     chunk.Hash,
				}).Errorf("Error writing to tempfile: %v", err)
				continue
			}
			conf.cache.Store(tempfilepath, structs.OpenTempFile{
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
		cache:   sync.Map{},
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, conf)
	}
	go Manager(ctx, conf)
}
