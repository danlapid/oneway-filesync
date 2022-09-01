package watcher

import (
	"context"
	"oneway-filesync/pkg/config"
	"oneway-filesync/pkg/database"
	"os"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), err
}

type watcherConfig struct {
	db    *gorm.DB
	input chan notify.EventInfo
	cache map[string]time.Time
}

// To save up on resources we only send files that haven't changed for the past 30 seconds
// otherwise many consecutive small changes will cause a large overhead on the sender/receiver
func worker(ctx context.Context, conf *watcherConfig) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			notify.Stop(conf.input)
			return
		case ei := <-conf.input:
			isdir, err := isDirectory(ei.Path())
			if err == nil && !isdir {
				conf.cache[ei.Path()] = time.Now()
				logrus.Infof("Noticed change in file '%s'", ei.Path())
			}
		case <-ticker.C:
			for path, lastupdated := range conf.cache {
				if time.Since(lastupdated).Seconds() > 30 {
					delete(conf.cache, path)
					err := database.QueueFileForSending(conf.db, path)
					if err != nil {
						logrus.Errorf("%v", err)
					} else {
						logrus.Infof("File '%s' queued for sending", path)
					}
				}
			}
		}
	}
}

func CreateWatcher(ctx context.Context, db *gorm.DB, watchdir string, input chan notify.EventInfo) {
	if err := notify.Watch(filepath.Join(watchdir, "..."), input, notify.Write, notify.Create); err != nil {
		logrus.Fatalf("%v", err)
	}
	conf := watcherConfig{
		db:    db,
		input: input,
		cache: make(map[string]time.Time),
	}
	go worker(ctx, &conf)
}

func Watcher(ctx context.Context, db *gorm.DB, conf config.Config) {
	events := make(chan notify.EventInfo, 500)

	CreateWatcher(ctx, db, conf.WatchDir, events)
}
