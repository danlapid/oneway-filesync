package database

import (
	"os"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestOpenDatabase(t *testing.T) {
	type args struct {
		tableprefix string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test-works", args{"t_"}, false},
		{"test-nullbyte", args{"t_\x00"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := OpenDatabase(tt.args.tableprefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenDatabase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			_ = os.Remove(strings.Split(DBFILE, "?")[0])
		})
	}
}

func TestClearDatabase(t *testing.T) {
	db1, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err = configureDatabase(db1); err != nil {
		t.Fatal(err)
	}
	db2, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		db *gorm.DB
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test-working", args{db1}, false},
		{"test-test-no-such-table", args{db2}, true},
		// {"test-test-null-db", args{&gorm.DB{}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ClearDatabase(tt.args.db); (err != nil) != tt.wantErr {
				t.Errorf("ClearDatabase() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQueueFileForSending(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err = configureDatabase(db); err != nil {
		t.Fatal(err)
	}
	type args struct {
		db        *gorm.DB
		path      string
		encrypted bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test-works", args{db, "a", false}, false},
		{"test-no-such-file", args{db, "a", false}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != "test-no-such-file" && tt.name != "test-patherror" {
				if err := os.WriteFile(tt.args.path, make([]byte, 4), os.ModePerm); err != nil {
					t.Fatal(err)
				}
			}
			defer os.Remove(tt.args.path)
			if err := QueueFileForSending(tt.args.db, tt.args.path, tt.args.encrypted); (err != nil) != tt.wantErr {
				t.Errorf("QueueFileForSending() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
