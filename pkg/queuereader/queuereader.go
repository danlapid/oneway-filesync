// Reads file paths to be sent from the queue
package queuereader

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/database"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type queueReaderConfig struct {
	db     *gorm.DB
	output chan database.File
}

func worker(ctx context.Context, conf *queueReaderConfig) {
	ticker := time.NewTicker(300 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var files []database.File
			conf.db.Where("Started = ? AND Finished = ?", false, false).Limit(100).Find(&files)
			for _, file := range files {
				file.Started = true
				err := conf.db.Save(&file).Error
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"Path": file.Path,
						"Hash": fmt.Sprintf("%x", file.Hash),
					}).Errorf("Error setting to Started in database %v", err)
					continue
				}
				conf.output <- file
			}
		}
	}
}

func CreateQueueReader(ctx context.Context, db *gorm.DB, output chan database.File) {
	conf := queueReaderConfig{
		db:     db,
		output: output,
	}
	go worker(ctx, &conf)
}
