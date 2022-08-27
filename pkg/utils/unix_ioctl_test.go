//go:build linux || darwin

package utils_test

import (
	"fmt"
	"net"
	"oneway-filesync/pkg/utils"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

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
		{"test1", args{1024 * 1024}, 8 * 1024 * 1024, false},
		{"test1", args{64 * 1024 * 1024}, 64 * 1024 * 1024, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn.SetReadBuffer(tt.args.bufsize)
			got, err := utils.GetReadBuffer(rawconn)
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

	sending_conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		logrus.Errorf("Error creating udp socket: %v", err)
		return
	}

	rawconn, err := receiving_conn.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}

	chunksize := 8192
	chunk := make([]byte, chunksize)
	for i := 0; i < 10; i++ {
		expected := (i + 1) * chunksize
		sending_conn.Write(chunk)
		time.Sleep(300 * time.Millisecond)
		avail, err := utils.GetAvailableBytes(rawconn)
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