package filecloser

import (
	"context"
	"oneway-filesync/pkg/structs"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"
)

func Test_normalizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"test1", "/tmp/out/check", "tmp/out/check"},
		{"test2", "c:\\tmp\\out\\check", "c/tmp/out/check"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizePath(tt.path); got != tt.want {
				t.Errorf("normalizePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_closeFile(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	hash := [32]byte{0x9f, 0x64, 0xa7, 0x47, 0xe1, 0xb9, 0x7f, 0x13, 0x1f, 0xab, 0xb6, 0xb4, 0x47, 0x29, 0x6c, 0x9b, 0x6f, 0x02, 0x01, 0xe7, 0x9f, 0xb3, 0xc5, 0x35, 0x6e, 0x6c, 0x77, 0xe8, 0x9b, 0x6a, 0x80, 0x6a}
	wronghash := hash
	wronghash[0] = 0

	type args struct {
		file   *structs.OpenTempFile
		outdir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test-works", args{&structs.OpenTempFile{TempFile: "a", Path: "b", Hash: hash, Encrypted: false, LastUpdated: time.Now()}, "out"}, false},
		{"test-hash-mismsatch", args{&structs.OpenTempFile{TempFile: "a", Path: "b", Hash: wronghash, Encrypted: false, LastUpdated: time.Now()}, "out"}, true},
		{"test-no-such-file", args{&structs.OpenTempFile{TempFile: "/tmp/adsasdasdsadas/adadsada/a", Path: "b", Hash: hash, Encrypted: false, LastUpdated: time.Now()}, "out"}, true},
		{"test-rename-fail", args{&structs.OpenTempFile{TempFile: "a", Path: "b\x00", Hash: hash, Encrypted: false, LastUpdated: time.Now()}, "out"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.RemoveAll(tt.args.outdir)
			if tt.name != "test-no-such-file" {
				if err := os.WriteFile(tt.args.file.TempFile, data, os.ModePerm); err != nil {
					t.Fatal(err)
				}
			}
			defer os.Remove(tt.args.file.TempFile)

			if err := closeFile(tt.args.file, tt.args.outdir); (err != nil) != tt.wantErr {
				t.Errorf("closeFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_worker(t *testing.T) {
	type args struct {
		ctx  context.Context
		conf *fileCloserConfig
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

func TestCreateFileCloser(t *testing.T) {
	type args struct {
		ctx         context.Context
		db          *gorm.DB
		outdir      string
		input       chan *structs.OpenTempFile
		workercount int
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CreateFileCloser(tt.args.ctx, tt.args.db, tt.args.outdir, tt.args.input, tt.args.workercount)
		})
	}
}
