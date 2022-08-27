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
				if err := conf.db.Save(&dbentry).Error; err != nil {
					logrus.WithFields(logrus.Fields{
						"TempFile":        file.TempFile,
						"Path":            file.Path,
						"Hash":            fmt.Sprintf("%x", file.Hash),
						"TransferSuccess": dbentry.Success,
					}).Errorf("Failed committing to db: %v", err)
				}
				continue
			}

			hash, err := structs.HashFile(f)
			err2 := f.Close()
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": file.TempFile,
					"Path":     file.Path,
					"Hash":     fmt.Sprintf("%x", file.Hash),
				}).Errorf("Error hashing tempfile: %v", err)
				dbentry.Success = false
				if err := conf.db.Save(&dbentry).Error; err != nil {
					logrus.WithFields(logrus.Fields{
						"TempFile":        file.TempFile,
						"Path":            file.Path,
						"Hash":            fmt.Sprintf("%x", file.Hash),
						"TempFileHash":    fmt.Sprintf("%x", hash),
						"TransferSuccess": dbentry.Success,
					}).Errorf("Failed committing to db: %v", err)
				}
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
				if err := conf.db.Save(&dbentry).Error; err != nil {
					logrus.WithFields(logrus.Fields{
						"TempFile":        file.TempFile,
						"Path":            file.Path,
						"Hash":            fmt.Sprintf("%x", file.Hash),
						"TempFileHash":    fmt.Sprintf("%x", hash),
						"TransferSuccess": dbentry.Success,
					}).Errorf("Failed committing to db: %v", err)
				}
				continue
			}
			if err2 != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": file.TempFile,
					"Path":     file.Path,
					"Hash":     fmt.Sprintf("%x", file.Hash),
				}).Errorf("Error ckisubg tempfile: %v", err)
			}

			newpath := filepath.Join(conf.outdir, normalizePath(file.Path))
			err = os.MkdirAll(filepath.Dir(newpath), os.ModePerm)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile":     file.TempFile,
					"Path":         file.Path,
					"Hash":         fmt.Sprintf("%x", file.Hash),
					"TempFileHash": fmt.Sprintf("%x", hash),
				}).Errorf("Failed creating directory path: %v", err)
				dbentry.Success = false
				if err := conf.db.Save(&dbentry).Error; err != nil {
					logrus.WithFields(logrus.Fields{
						"TempFile":        file.TempFile,
						"Path":            file.Path,
						"Hash":            fmt.Sprintf("%x", file.Hash),
						"TempFileHash":    fmt.Sprintf("%x", hash),
						"TransferSuccess": dbentry.Success,
					}).Errorf("Failed committing to db: %v", err)
				}
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
				if err := conf.db.Save(&dbentry).Error; err != nil {
					logrus.WithFields(logrus.Fields{
						"TempFile":        file.TempFile,
						"Path":            file.Path,
						"Hash":            fmt.Sprintf("%x", file.Hash),
						"TempFileHash":    fmt.Sprintf("%x", hash),
						"TransferSuccess": dbentry.Success,
					}).Errorf("Failed committing to db: %v", err)
				}
				continue
			}

			logrus.WithFields(logrus.Fields{
				"Path":    file.Path,
				"Hash":    fmt.Sprintf("%x", file.Hash),
				"NewPath": newpath,
			}).Infof("Successfully finished writing file")
			dbentry.Success = true
			if err := conf.db.Save(&dbentry).Error; err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile":        file.TempFile,
					"Path":            file.Path,
					"Hash":            fmt.Sprintf("%x", file.Hash),
					"TempFileHash":    fmt.Sprintf("%x", hash),
					"TransferSuccess": dbentry.Success,
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
