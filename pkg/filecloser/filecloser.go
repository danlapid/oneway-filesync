package filecloser

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/structs"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type FileCloser struct {
	db     *gorm.DB
	outdir string
	input  chan *structs.OpenTempFile
}

func worker(ctx context.Context, conf *FileCloser) {
	for {
		select {
		case <-ctx.Done():
			return
		case file := <-conf.input:
			dbentry := database.File{
				Path:     file.Path,
				Hash:     file.Hash[:],
				Started:  true,
				Finished: true,
			}

			f, err := os.Open(file.TempFile)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": file.TempFile,
					"Path":     file.Path,
					"Hash":     fmt.Sprintf("%x", file.Hash),
				}).Errorf("Error opening tempfile: %v", err)
				dbentry.Success = false
				conf.db.Save(&dbentry)
				continue
			}
			defer f.Close()

			hash, err := structs.HashFile(f)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": file.TempFile,
					"Path":     file.Path,
					"Hash":     fmt.Sprintf("%x", file.Hash),
				}).Errorf("Error hashing tempfile: %v", err)
				dbentry.Success = false
				conf.db.Save(&dbentry)
				continue
			}
			if hash != file.Hash {
				logrus.WithFields(logrus.Fields{
					"TempFile":     file.TempFile,
					"Path":         file.Path,
					"Hash":         fmt.Sprintf("%x", file.Hash),
					"TempFileHash": fmt.Sprintf("%x", hash),
				}).Errorf("Hash mismatch")
				dbentry.Success = false
				conf.db.Save(&dbentry)
				continue
			}

			newpath := filepath.Join(conf.outdir, file.Path)
			err = os.MkdirAll(filepath.Dir(newpath), os.ModePerm)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile":     file.TempFile,
					"Path":         file.Path,
					"Hash":         fmt.Sprintf("%x", file.Hash),
					"TempFileHash": fmt.Sprintf("%x", hash),
				}).Errorf("Failed creating directory path: %v", err)
				dbentry.Success = false
				conf.db.Save(&dbentry)
				continue
			}

			err = os.Rename(file.TempFile, newpath)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile":     file.TempFile,
					"Path":         file.Path,
					"Hash":         fmt.Sprintf("%x", file.Hash),
					"TempFileHash": fmt.Sprintf("%x", hash),
				}).Errorf("Failed moving tempfile to new location: %v", err)
				dbentry.Success = false
				conf.db.Save(&dbentry)
				continue
			}

			logrus.WithFields(logrus.Fields{
				"Path":    file.Path,
				"Hash":    fmt.Sprintf("%x", file.Hash),
				"NewPath": newpath,
			}).Infof("Successfully finished writing file")
			dbentry.Success = true
			conf.db.Save(&dbentry)
		}
	}
}

func CreateFileCloser(ctx context.Context, db *gorm.DB, outdir string, input chan *structs.OpenTempFile, workercount int) {
	conf := FileCloser{
		db:     db,
		outdir: outdir,
		input:  input,
	}
	for i := 0; i < workercount; i++ {
		go worker(ctx, &conf)
	}
}
