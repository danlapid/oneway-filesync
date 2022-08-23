package filecloser

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/structs"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type FileCloser struct {
	outdir string
	input  chan structs.OpenTempFile
}

func Worker(ctx context.Context, conf *FileCloser) {
	db, err := database.OpenDatabase()
	if err != nil {
		logrus.Errorf("Error connecting to the database: %v", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case file := <-conf.input:
			dbentry := database.File{
				Path:     file.Path,
				Size:     file.Size,
				Hash:     file.Hash[:],
				Started:  true,
				Finished: true,
			}

			err := os.Truncate(file.TempFile, file.Size)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": file.TempFile,
					"Path":     file.Path,
					"Hash":     fmt.Sprintf("%x", file.Hash),
				}).Errorf("Error truncating tempfile to original size: %v", err)
				dbentry.Success = false
				db.Save(&dbentry)
				continue
			}

			f, err := os.Open(file.TempFile)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile": file.TempFile,
					"Path":     file.Path,
					"Hash":     fmt.Sprintf("%x", file.Hash),
				}).Errorf("Error opening tempfile: %v", err)
				dbentry.Success = false
				db.Save(&dbentry)
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
				db.Save(&dbentry)
				continue
			}
			if hash != file.Hash {
				logrus.WithFields(logrus.Fields{
					"TempFile":     file.TempFile,
					"Path":         file.Path,
					"Hash":         fmt.Sprintf("%x", file.Hash),
					"TempFileHash": hash,
				}).Errorf("Hash mismatch ", err)
				dbentry.Success = false
				db.Save(&dbentry)
				continue
			}

			newpath := filepath.Join(conf.outdir, file.Path)
			err = os.MkdirAll(filepath.Dir(newpath), os.ModePerm)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile":     file.TempFile,
					"Path":         file.Path,
					"Hash":         fmt.Sprintf("%x", file.Hash),
					"TempFileHash": hash,
				}).Errorf("Failed creating directory path: %v", err)
				dbentry.Success = false
				db.Save(&dbentry)
				continue
			}

			err = os.Rename(file.TempFile, newpath)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"TempFile":     file.TempFile,
					"Path":         file.Path,
					"Hash":         fmt.Sprintf("%x", file.Hash),
					"TempFileHash": hash,
				}).Errorf("Failed moving tempfile to new location: %v", err)
				dbentry.Success = false
				db.Save(&dbentry)
				continue
			}

			logrus.WithFields(logrus.Fields{
				"Path":    file.Path,
				"Hash":    fmt.Sprintf("%x", file.Hash),
				"NewPath": newpath,
			}).Infof("Successfully finished writing file")
			dbentry.Success = true
			db.Save(&dbentry)
		}
	}
}

func CreateFileCloser(ctx context.Context, outdir string, input chan structs.OpenTempFile, workercount int) {
	conf := FileCloser{
		outdir: outdir,
		input:  input,
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, &conf)
	}
}
