package filecloser

import (
	"context"
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

func Worker(ctx context.Context, conf FileCloser) {
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
					"Hash":     file.Hash,
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
					"Hash":     file.Hash,
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
					"Hash":     file.Hash,
				}).Errorf("Error hashing tempfile: %v", err)
				dbentry.Success = false
				db.Save(&dbentry)
				continue
			}
			if hash != file.Hash {
				logrus.WithFields(logrus.Fields{
					"TempFile":     file.TempFile,
					"Path":         file.Path,
					"Hash":         file.Hash,
					"TempFileHash": hash,
				}).Errorf("Hash mismatch ", err)
				dbentry.Success = false
				db.Save(&dbentry)
				continue
			}

			newpath := filepath.Join(conf.outdir, file.Path)
			os.Rename(file.TempFile, newpath)
			logrus.WithFields(logrus.Fields{
				"Path":    file.Path,
				"Hash":    file.Hash,
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
		go Worker(ctx, conf)
	}
}
