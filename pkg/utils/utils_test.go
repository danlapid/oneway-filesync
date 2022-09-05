package utils

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
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

func TestInitializeLogging(t *testing.T) {
	type args struct {
		logFile string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"test1", args{"logfile.txt"}, false},
		{"test-invalid-path", args{"/tmp/asuhdaiusfa/asdada/logfile.txt"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitializeLogging(tt.args.logFile)
			if tt.wantErr {
				return
			}
			defer os.Remove(tt.args.logFile)

			logrus.Info("Test")
			data, err := os.ReadFile(tt.args.logFile)
			if err != nil {
				t.Fatal("logfile did not create")
			}
			if !(strings.Contains(string(data), "Test") && strings.Contains(string(data), "utils_test.go")) {
				t.Fatal("logging failed")
			}
		})
	}
}

func TestMap(t *testing.T) {
	m := RWMutexMap[int, string]{}
	m.Store(1, "a")
	if v, ok := m.Load(1); !ok || v != "a" {
		t.Fatal("Store then load failed")
	}

	if actual, loaded := m.LoadOrStore(1, "b"); !loaded || actual != "a" {
		t.Fatal("LoadOrStore-Load failed")
	}
	if actual, loaded := m.LoadOrStore(2, "b"); loaded || actual != "b" {
		t.Fatal("LoadOrStore-Store failed")
	}

	if value, loaded := m.LoadAndDelete(2); !loaded || value != "b" {
		t.Fatal("LoadAndDelete-Exists failed")
	}
	if _, loaded := m.LoadAndDelete(2); loaded {
		t.Fatal("LoadAndDelete-NotExists failed")
	}

	m.Delete(1)
	m.Store(1, "a")
	m.Store(2, "b")
	m2 := map[int]string{}
	m2[1] = "a"
	m2[2] = "b"
	m.Range(func(key int, value string) bool {
		if m2[key] != value {
			t.Fatal("mismating value in range")
		}
		if key == 1 {
			return true
		} else {
			return false
		}
	})

	if m.Len() != 2 {
		t.Fatal("Len failed")
	}
}

// Removed CtrlC test due to: https://github.com/golang/go/issues/46354
func TestCtrlC(t *testing.T) {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		ch := CtrlC()
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatal(err)
		}
		err = p.Signal(os.Interrupt)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := <-ch
		if !ok {
			t.Fatal("Ctrl c not caught")
		}

	}
}
