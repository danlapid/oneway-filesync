package structs_test

import (
	"oneway-filesync/pkg/structs"
	"reflect"
	"testing"
)

func TestChunk(t *testing.T) {
	type args struct {
		data structs.Chunk
	}
	tests := []struct {
		name string
		args args
	}{
		{"test", args{structs.Chunk{
			Path:        "/tmp/abc",
			Encrypted:   true,
			DataOffset:  17124124,
			DataPadding: 5,
			ShareIndex:  4,
			Data:        make([]byte, 3000),
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := tt.args.data.Encode()
			if err != nil {
				t.Errorf("Encode() error = %v", err)
				return
			}
			got, err := structs.DecodeChunk(buf)
			if err != nil {
				t.Errorf("DecodeChunk() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.args.data) {
				t.Errorf("DecodeChunk() = %v, want %v", got, tt.args.data)
			}
		})
	}
}
