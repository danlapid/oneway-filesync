package filecloser

import (
	"context"
	"errors"
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
	l := logrus.WithFields(logrus.Fields{
		"TempFile": file.TempFile,
		"Path":     file.Path,
		"Hash":     fmt.Sprintf("%x", file.Hash),
	})

	f, err := os.Open(file.TempFile)
	if err != nil {
		l.Errorf("Error opening tempfile: %v", err)
		return err
	}

	hash, err := structs.HashFile(f, false)
	err2 := f.Close()
	if err != nil {
		l.Errorf("Error hashing tempfile: %v", err)
		return err
	}
	if err2 != nil {
		l.Errorf("Error closing tempfile: %v", err2)
		// Not returning error on purpose
	}
	if hash != file.Hash {
		l.WithField("TempFileHash", fmt.Sprintf("%x", hash)).Errorf("Hash mismatch")
		return errors.New("hash mismatch")
	}

	newpath := filepath.Join(outdir, normalizePath(file.Path))
	if file.Encrypted {
		newpath += ".zip"
	}
	err = os.MkdirAll(filepath.Dir(newpath), os.ModePerm)
	if err != nil {
		l.Errorf("Failed creating directory path: %v", err)
		return err
	}

	err = os.Rename(file.TempFile, newpath)
	if err != nil {
		l.Errorf("Failed moving tempfile to new location: %v", err)
		return err
	}

	l.WithField("NewPath", newpath).Infof("Successfully finished writing file")
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
			} else {
				dbentry.Success = true
			}
			if err := conf.db.Save(&dbentry).Error; err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": file.TempFile,
					"Path":     file.Path,
					"Hash":     fmt.Sprintf("%x", file.Hash),
				}).Errorf("Failed committing to db: %v", err)
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
