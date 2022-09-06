package udpsender

import (
	"bytes"
	"context"
	"crypto/rand"
	"math/big"
	"net"
	"oneway-filesync/pkg/structs"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func randint(max int64) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		panic(err)
	}
	return int(nBig.Int64())
}

func Test_worker(t *testing.T) {
	ip := "127.0.0.1"
	port := randint(30000) + 30000
	addr := net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	receiving_conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		t.Fatal(err)
	}
	defer receiving_conn.Close()

	type args struct {
		ip    string
		port  int
		chunk structs.Chunk
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr string
	}{
		{"test-works", args{ip, port, structs.Chunk{}}, false, ""},
		{"test-socket-err", args{ip, 88888, structs.Chunk{}}, true, "Error creating udp socket"},
		{"test-message-too-long", args{ip, port, structs.Chunk{Data: make([]byte, 100*1024)}}, true, "write: message too long"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var memLog bytes.Buffer
			logrus.SetOutput(&memLog)

			input := make(chan *structs.Chunk, 5)
			input <- &tt.args.chunk
			conf := udpSenderConfig{tt.args.ip, tt.args.port, input}
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(2 * time.Second)
				cancel()
			}()
			worker(ctx, &conf)

			if tt.wantErr {
				if !strings.Contains(memLog.String(), tt.expectedErr) {
					t.Fatalf("Expected not in log, '%v' not in '%v'", tt.expectedErr, memLog.String())
				}
			} else {
				receiving_conn.Read(make([]byte, 8192))
			}
		})
	}
}
