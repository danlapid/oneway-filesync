package watcher

import (
	"context"
	"oneway-filesync/pkg/database"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type watcherConfig struct {
	db    *gorm.DB
	input chan notify.EventInfo
	cache map[string]time.Time
}

// To save up on resources we only send files that haven't changed for the past 60 seconds
// otherwise many consecutive small changes will cause a large overhead on the sender/receiver
func worker(ctx context.Context, conf *watcherConfig) {
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-ctx.Done():
			notify.Stop(conf.input)
			return
		case ei := <-conf.input:
			conf.cache[ei.Path()] = time.Now()
		case <-ticker.C:
			for path, lastupdated := range conf.cache {
				if time.Since(lastupdated).Seconds() > 60 {
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
		logrus.Errorf("%v", err)
		return
	}
	conf := watcherConfig{
		db:    db,
		input: input,
		cache: make(map[string]time.Time),
	}
	go worker(ctx, &conf)
}
