package config_test

import (
	"oneway-filesync/pkg/config"
	"os"
	"reflect"
	"testing"
)

func TestGetConfig(t *testing.T) {
	type args struct {
		configtext string
	}
	tests := []struct {
		name    string
		args    args
		want    config.Config
		wantErr bool
	}{
		{
			name: "test1",
			args: args{configtext: `
				ReceiverIP = "127.0.0.1"
				ReceiverPort = 5000
				BandwidthLimit = 10000000
				ChunkSize = 8192
				ChunkFecRequired = 5
				ChunkFecTotal = 10
				OutDir = "./out"
				WatchDir = "./tmp"`},
			want: config.Config{
				ReceiverIP:       "127.0.0.1",
				ReceiverPort:     5000,
				BandwidthLimit:   10000000,
				ChunkSize:        8192,
				ChunkFecRequired: 5,
				ChunkFecTotal:    10,
				OutDir:           "./out",
				WatchDir:         "./tmp",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filename string
			func() {
				f, err := os.CreateTemp("", "")
				if (err != nil) != tt.wantErr {
					t.Errorf("CreateTemp() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				defer f.Close()
				filename = f.Name()
				_, err = f.WriteString(tt.args.configtext)
				if (err != nil) != tt.wantErr {
					t.Errorf("WriteString() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}()
			defer os.Remove(filename)
			got, err := config.GetConfig(filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
