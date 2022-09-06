package filecloser

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/structs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func normalizePath(path string) string {
	newpath := strings.ReplaceAll(path, ":", "")
	if strings.Contains(newpath, "\\") {
		return filepath.Join(strings.Split(newpath, "\\")...)
	} else {

		return filepath.Join(strings.Split(newpath, "/")...)
	}
}
func closeFile(file *structs.OpenTempFile, outdir string) error {
	f, err := os.Open(file.TempFile)
	if err != nil {
		return fmt.Errorf("error opening tempfile: %v", err)
	}

	hash, err := structs.HashFile(f, false)
	_ = f.Close() // Ignoring error on purpose
	if err != nil {
		return fmt.Errorf("error hashing tempfile: %v", err)
	}

	if hash != file.Hash {
		return fmt.Errorf("hash mismatch '%v'!='%v'", fmt.Sprintf("%x", hash), fmt.Sprintf("%x", file.Hash))
	}

	newpath := filepath.Join(outdir, normalizePath(file.Path))
	if file.Encrypted {
		newpath += ".zip"
	}
	err = os.MkdirAll(filepath.Dir(newpath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed creating directory path: %v", err)
	}

	err = os.Rename(file.TempFile, newpath)
	if err != nil {
		return fmt.Errorf("failed moving tempfile to new location: %v", err)
	}

	return nil
}

type fileCloserConfig struct {
	db     *gorm.DB
	outdir string
	input  chan *structs.OpenTempFile
}

func worker(ctx context.Context, conf *fileCloserConfig) {
	for {
		select {
		case <-ctx.Done():
			return
		case file := <-conf.input:
			l := logrus.WithFields(logrus.Fields{
				"TempFile": file.TempFile,
				"Path":     file.Path,
				"Hash":     fmt.Sprintf("%x", file.Hash),
			})
			dbentry := database.File{
				Path:      file.Path,
				Hash:      file.Hash[:],
				Encrypted: file.Encrypted,
				Started:   true,
				Finished:  true,
			}

			err := closeFile(file, conf.outdir)
			if err != nil {
				dbentry.Success = false
				l.Error(err)
			} else {
				dbentry.Success = true
				l.Infof("Successfully finished writing file")
			}
			if err := conf.db.Save(&dbentry).Error; err != nil {
				l.Errorf("Failed committing to db: %v", err)
			}
		}
	}
}

func CreateFileCloser(ctx context.Context, db *gorm.DB, outdir string, input chan *structs.OpenTempFile, workercount int) {
	conf := fileCloserConfig{
		db:     db,
		outdir: outdir,
		input:  input,
	}
	for i := 0; i < workercount; i++ {
		go worker(ctx, &conf)
	}
}
