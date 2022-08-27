//go:build windows

package utils_test

import (
	"oneway-filesync/pkg/utils"
	"syscall"
	"testing"
)

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
