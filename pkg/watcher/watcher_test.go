package watcher

import (
	"bytes"
	"context"
	"oneway-filesync/pkg/config"
	"os"
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

func Test_worker(t *testing.T) {
	type args struct {
		ctx  context.Context
		conf *watcherConfig
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker(tt.args.ctx, tt.args.conf)
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

func TestCreateWatcher_baddb(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	var memLog bytes.Buffer
	logrus.SetOutput(&memLog)

	ctx, cancel := context.WithCancel(context.Background())
	CreateWatcher(ctx, db, ".", false, make(chan notify.EventInfo, 5))

	err = os.WriteFile("testfile", make([]byte, 20), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("testfile")
	time.Sleep(60 * time.Second)
	cancel()

	if !strings.Contains(memLog.String(), "Failed to queue file for sending:") {
		t.Fatalf("Expected not in log, '%v' not in '%v'", "Failed to queue file for sending:", memLog.String())
	}
}

func TestWatcher(t *testing.T) {
	type args struct {
		ctx  context.Context
		db   *gorm.DB
		conf config.Config
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Watcher(tt.args.ctx, tt.args.db, tt.args.conf)
		})
	}
}
