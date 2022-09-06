package udpreceiver

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"oneway-filesync/pkg/structs"

	"github.com/sirupsen/logrus"
)

func randint(max int64) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		panic(err)
	}
	return int(nBig.Int64())
}

func Test_manager(t *testing.T) {
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

	sending_conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		t.Fatal(err)
	}
	defer sending_conn.Close()
	chunksize := 8192
	chunk := make([]byte, chunksize)
	err = receiving_conn.SetReadBuffer(5 * chunksize)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		_, err := sending_conn.Write(chunk)
		if err != nil {
			t.Fatal(err)
		}
	}

	type args struct {
		conf *udpReceiverConfig
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{"test-invalid-socket", args{&udpReceiverConfig{&net.UDPConn{}, 8192, make(chan *structs.Chunk)}}, "Error getting raw socket"},
		{"test-buffers-full", args{&udpReceiverConfig{receiving_conn, 8192, make(chan *structs.Chunk)}}, "Buffers are filling up loss of data is probable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var memLog bytes.Buffer
			logrus.SetOutput(&memLog)

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(2 * time.Second)
				cancel()
			}()
			manager(ctx, tt.args.conf)

			if !strings.Contains(memLog.String(), tt.expected) {
				t.Fatalf("Expected not in log, '%v' not in '%v'", tt.expected, memLog.String())
			}
		})
	}
}

func Test_worker_close_conn(t *testing.T) {
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

	chunksize := 8192

	output := make(chan *structs.Chunk, 5)
	type args struct {
		conf *udpReceiverConfig
	}
	tests := []struct {
		name string
		args args
	}{
		{"test1", args{&udpReceiverConfig{receiving_conn, chunksize, output}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(2 * time.Second)
				tt.args.conf.conn.Close()
				cancel()
			}()
			worker(ctx, tt.args.conf)
		})
	}
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

	sending_conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		t.Fatal(err)
	}
	defer sending_conn.Close()

	chunksize := 8192

	output := make(chan *structs.Chunk, 5)
	conf := &udpReceiverConfig{receiving_conn, chunksize, output}
	chunk := structs.Chunk{Path: "a", Data: make([]byte, chunksize/2)}

	var memLog bytes.Buffer
	logrus.SetOutput(&memLog)

	data, err := chunk.Encode()
	if err != nil {
		t.Fatal(err)
	}
	_, err = sending_conn.Write(data)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(2 * time.Second)
		conf.conn.Close()
		cancel()
	}()
	worker(ctx, conf)

	got := <-output
	if !reflect.DeepEqual(*got, chunk) {
		t.Fatalf("DecodeChunk() = %v, want %v", got, chunk)
	}
}

func Test_worker_error_invalid_socket(t *testing.T) {
	chunksize := 8192
	output := make(chan *structs.Chunk, 5)
	conf := &udpReceiverConfig{&net.UDPConn{}, chunksize, output}

	var memLog bytes.Buffer
	logrus.SetOutput(&memLog)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(2 * time.Second)
		conf.conn.Close()
		cancel()
	}()
	worker(ctx, conf)

	if !strings.Contains(memLog.String(), "Error reading from socket") {
		t.Fatalf("Expected not in log, '%v' not in '%v'", "Error reading from socket", memLog.String())
	}
}

func Test_worker_error_decoding(t *testing.T) {
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

	sending_conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		t.Fatal(err)
	}
	defer sending_conn.Close()

	chunksize := 8192
	output := make(chan *structs.Chunk, 5)
	conf := &udpReceiverConfig{receiving_conn, chunksize, output}
	data := make([]byte, chunksize/2)
	for i := range data {
		data[i] = 0xff
	}
	var memLog bytes.Buffer
	logrus.SetOutput(&memLog)

	_, err = sending_conn.Write(data)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(2 * time.Second)
		conf.conn.Close()
		cancel()
	}()
	worker(ctx, conf)

	if !strings.Contains(memLog.String(), "Error decoding chunk") {
		t.Fatalf("Expected not in log, '%v' not in '%v'", "Error decoding chunk", memLog.String())
	}
}

func TestCreateUdpReceiver(t *testing.T) {
	// fail create socket test
	var memLog bytes.Buffer
	logrus.SetOutput(&memLog)
	ctx, cancel := context.WithCancel(context.Background())
	CreateUdpReceiver(ctx, "127.0.0.1", 88888, 8192, make(chan *structs.Chunk), 1)
	cancel()
	if !strings.Contains(memLog.String(), "Error creating udp socket") {
		t.Fatalf("Expected not in log, '%v' not in '%v'", "Error creating udp socket", memLog.String())
	}
}
