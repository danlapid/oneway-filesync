package utils

import (
	"os"
	"syscall"
	"testing"
)

func Test_formatFilePath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"test1", args{"/a/b/c/d.tmp"}, "d.tmp"},
		{"test2", args{"C:\\a\\b\\c\\d.tmp"}, "d.tmp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatFilePath(tt.args.path); got != tt.want {
				t.Errorf("formatFilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCtrlC(t *testing.T) {
	ch := CtrlC()
	err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := <-ch
	if !ok {
		t.Fatal("Ctrl c not caught")
	}
}

func TestInitializeLogging(t *testing.T) {
	type args struct {
		logFile string
	}
	tests := []struct {
		name string
		args args
	}{
		{"test1", args{"logfile.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitializeLogging(tt.args.logFile)
			if _, err := os.Stat(tt.args.logFile); os.IsExist(err) {
				t.Fatal("logfile did not create")
			}
			os.Remove(tt.args.logFile)
		})
	}
}
