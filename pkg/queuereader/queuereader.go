// Reads file paths to be sent from the queue
package queuereader

import (
	"context"
	"fmt"
	"oneway-filesync/pkg/database"
	"time"

	"github.com/sirupsen/logrus"
)

type QueueReader struct {
	output chan database.File
}

func Worker(ctx context.Context, conf *QueueReader) {
	db, err := database.OpenDatabase()
	if err != nil {
		logrus.Errorf("Error connecting to the database: %v", err)
		return
	}
	ticker := time.NewTicker(300 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var files []database.File
			db.Where("Started = ? AND Finished = ?", false, false).Limit(100).Find(&files)
			for _, file := range files {
				file.Started = true
				err := db.Save(&file).Error
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

func CreateQueueReader(ctx context.Context, output chan database.File) {
	conf := QueueReader{
		output: output,
	}
	go Worker(ctx, &conf)
}
