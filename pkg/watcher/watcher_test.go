package watcher

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func Test_isDirectory(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"test-works", args{"."}, true, false},
		{"test-non-existent", args{"nonexistentdir"}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isDirectory(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("isDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isDirectory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateWatcher_baddir(t *testing.T) {
	var memLog bytes.Buffer
	logrus.SetOutput(&memLog)

	ctx, cancel := context.WithCancel(context.Background())
	CreateWatcher(ctx, &gorm.DB{}, "nonexistentdir", false, make(chan notify.EventInfo, 5))
	cancel()

	if !strings.Contains(memLog.String(), "Failed to watch dir with error") {
		t.Fatalf("Expected not in log, '%v' not in '%v'", "Failed to watch dir with error", memLog.String())
	}
}

func Test_worker_baddb(t *testing.T) {
	var memLog bytes.Buffer
	logrus.SetOutput(&memLog)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Fatal(err)
	}

	conf := watcherConfig{
		db:        db,
		encrypted: false,
		input:     make(chan notify.EventInfo, 5),
		cache:     make(map[string]time.Time),
	}

	if err := notify.Watch(filepath.Join(".", "..."), conf.input, notify.Write, notify.Create); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err = os.WriteFile("testfile", make([]byte, 20), os.ModePerm)
		if err != nil {
			t.Error(err)
		}
		defer os.Remove("testfile")
		time.Sleep(60 * time.Second)
		cancel()
	}()
	worker(ctx, &conf)

	if !strings.Contains(memLog.String(), "Failed to queue file for sending:") {
		t.Fatalf("Expected not in log, '%v' not in '%v'", "Failed to queue file for sending:", memLog.String())
	}
}
