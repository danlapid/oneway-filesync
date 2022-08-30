package utils

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

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

func TestCtrlC(t *testing.T) {
	ch := CtrlC()
	err := sendCtrlC(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	_, ok := <-ch
	if !ok {
		t.Fatal("Ctrl c not caught")
	}
}

func TestGetReadBuffer(t *testing.T) {
	ip := "127.0.0.1"
	port := 54249
	addr := net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	rawconn, err := conn.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		bufsize int
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{"test1", args{8 * 1024}, 8 * 1024, false},
		{"test2", args{1024 * 1024}, 1024 * 1024, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := conn.SetReadBuffer(tt.args.bufsize)
			if err != nil {
				t.Error(err)
				return
			}
			got, err := GetReadBuffer(rawconn)
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
	ip := "127.0.0.1"
	port := 54249
	addr := net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	receiving_conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		t.Fatal(err)
	}
	defer receiving_conn.Close()

	err = receiving_conn.SetReadBuffer(8192 * 10)
	if err != nil {
		t.Fatal(err)
	}

	sending_conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		logrus.Errorf("Error creating udp socket: %v", err)
		return
	}
	defer sending_conn.Close()

	rawconn, err := receiving_conn.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}

	chunksize := 8192
	chunk := make([]byte, chunksize)
	for i := 0; i < 10; i++ {
		expected := (i + 1) * chunksize
		_, err := sending_conn.Write(chunk)
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(300 * time.Millisecond)
		avail, err := GetAvailableBytes(rawconn)
		if err != nil {
			t.Errorf("GetAvailableBytes() error = %v", err)
			return
		}
		if avail < expected {
			t.Errorf("GetAvailableBytes() = %v, want %v", avail, expected)
			return
		}
	}
}
