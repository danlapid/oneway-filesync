//go:build windows

package utils_test

import (
	"oneway-filesync/pkg/utils"
	"syscall"
	"testing"
)

func sendCtrlC(t *testing.T, pid int) {
	d, e := syscall.LoadDLL("kernel32.dll")
	if e != nil {
		t.Fatalf("LoadDLL: %v\n", e)
	}
	p, e := d.FindProc("GenerateConsoleCtrlEvent")
	if e != nil {
		t.Fatalf("FindProc: %v\n", e)
	}
	r, _, e := p.Call(syscall.CTRL_C_EVENT, uintptr(pid))
	if r == 0 {
		t.Fatalf("GenerateConsoleCtrlEvent: %v\n", e)
	}
}

func TestCtrlC(t *testing.T) {
	ch := CtrlC()
	sendCtrlC(t, os.Getpid())
	_, ok := <-ch
	if !ok {
		t.Fatal("Ctrl c not caught")
	}
}

func TestGetReadBuffer(t *testing.T) {
	type args struct {
		rawconn syscall.RawConn
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.GetReadBuffer(tt.args.rawconn)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReadBuffer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetReadBuffer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAvailableBytes(t *testing.T) {
	type args struct {
		rawconn syscall.RawConn
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := utils.GetAvailableBytes(tt.args.rawconn)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAvailableBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetAvailableBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
