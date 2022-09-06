package filereader

import (
	"bytes"
	"context"
	"oneway-filesync/pkg/database"
	"oneway-filesync/pkg/structs"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func Test_sendfile(t *testing.T) {
	data := make([]byte, 4*8192)
	hash := []byte{0x9f, 0x64, 0xa7, 0x47, 0xe1, 0xb9, 0x7f, 0x13, 0x1f, 0xab, 0xb6, 0xb4, 0x47, 0x29, 0x6c, 0x9b, 0x6f, 0x02, 0x01, 0xe7, 0x9f, 0xb3, 0xc5, 0x35, 0x6e, 0x6c, 0x77, 0xe8, 0x9b, 0x6a, 0x80, 0x6a}
	type args struct {
		file *database.File
		conf *fileReaderConfig
	}
	tests := []struct {
		name     string
		args     args
		expected int
		wantErr  bool
	}{
		{"test-regular", args{
			file: &database.File{Path: "a", Hash: hash, Encrypted: false},
			conf: &fileReaderConfig{chunksize: 8192, required: 2},
		}, 3, false},
		{"test-encrypted", args{
			file: &database.File{Path: "a", Hash: hash, Encrypted: true},
			conf: &fileReaderConfig{chunksize: 8192, required: 2},
		}, 1, false},
		{"test-no-such-file", args{
			file: &database.File{Path: "b", Hash: hash, Encrypted: false},
			conf: &fileReaderConfig{chunksize: 8192, required: 2},
		}, 4, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := make(chan *structs.Chunk, 5)
			tt.args.conf.output = out

			if tt.name != "test-no-such-file" {
				if err := os.WriteFile(tt.args.file.Path, data, os.ModePerm); err != nil {
					t.Fatal(err)
				}
			}
			defer os.Remove(tt.args.file.Path)

			if err := sendfile(tt.args.file, tt.args.conf); (err != nil) != tt.wantErr {
				t.Fatalf("sendfile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if len(out) != tt.expected {
					t.Fatalf("Got too many chunks %v!=%v", len(out), tt.expected)
				}
			}
		})
	}
}

func Test_worker(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	hash := []byte{0x9f, 0x64, 0xa7, 0x47, 0xe1, 0xb9, 0x7f, 0x13, 0x1f, 0xab, 0xb6, 0xb4, 0x47, 0x29, 0x6c, 0x9b, 0x6f, 0x02, 0x01, 0xe7, 0x9f, 0xb3, 0xc5, 0x35, 0x6e, 0x6c, 0x77, 0xe8, 0x9b, 0x6a, 0x80, 0x6a}
	type args struct {
		file database.File
		conf *fileReaderConfig
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{"test-error-db", args{
			file: database.File{Path: "a", Hash: hash},
			conf: &fileReaderConfig{chunksize: 8192, required: 2},
		}, "Error updating Finished in database"},
		{"test-no-such-file", args{
			file: database.File{Path: "b", Hash: hash},
			conf: &fileReaderConfig{chunksize: 8192, required: 2},
		}, "File sending failed with err: error opening file:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var memLog bytes.Buffer
			logrus.SetOutput(&memLog)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Discard})
			if err != nil {
				t.Fatal(err)
			}
			tt.args.conf.db = db
			in := make(chan database.File, 5)
			tt.args.conf.input = in
			out := make(chan *structs.Chunk, 5)
			tt.args.conf.output = out

			if tt.name != "test-no-such-file" {
				if err := os.WriteFile(tt.args.file.Path, data, os.ModePerm); err != nil {
					t.Fatal(err)
				}
			}
			defer os.Remove(tt.args.file.Path)
			in <- tt.args.file

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(2 * time.Second)
				cancel()
			}()
			worker(ctx, tt.args.conf)

			if !strings.Contains(memLog.String(), tt.expected) {
				t.Fatalf("Expected not in log, '%v' not in '%v'", tt.expected, memLog.String())
			}
		})
	}
}
