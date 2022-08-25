package fecbench_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/klauspost/reedsolomon"
	"github.com/vivint/infectious"
)

func fillRandom(p []byte) {
	for i := 0; i < len(p); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(p) && j < 7; j++ {
			p[i+j] = byte(val)
			val >>= 8
		}
	}
}

func benchmarkReedSolomon(b *testing.B, enc reedsolomon.Encoder, data []byte) []byte {
	// Split the data into shards
	shards, err := enc.Split(data)
	if err != nil {
		b.Fatal(err)
	}

	// Encode the parity set
	err = enc.Encode(shards)
	if err != nil {
		b.Fatal(err)
	}

	// Reconstruct the shards
	shards[0], shards[1], shards[len(shards)-1], shards[len(shards)-2] = nil, nil, nil, nil
	err = enc.ReconstructData(shards)
	if err != nil {
		b.Fatal(err)
	}

	buf := make([]byte, len(shards[0])*4)
	for i, shard := range shards[:4] {
		copy(buf[i*len(shards[0]):], shard)
	}
	return buf
}

func BenchmarkReedSolomon(b *testing.B) {
	// Create some sample data
	datalen := 64 * 1024
	datanum := 64 * 1024
	data := make([]byte, datalen)
	b.ReportAllocs()
	b.SetBytes(int64(datalen * datanum))

	// Create an encoder with 4 data and 4 parity slices.
	enc, err := reedsolomon.New(4, 4)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.StopTimer()
	for i := 0; i < datanum; i++ {
		fillRandom(data)
		b.StartTimer()
		buf := benchmarkReedSolomon(b, enc, data)
		b.StopTimer()
		if !bytes.Equal(buf, data) {
			b.Fatal("recovered bytes do not match")
		}
	}
}

func benchmarkInfecious(b *testing.B, enc *infectious.FEC, data []byte) []byte {
	shares := make([]infectious.Share, enc.Total())
	for i := 0; i < enc.Total(); i++ {
		shares[i].Number = i
		shares[i].Data = make([]byte, len(data)/enc.Required())
		err := enc.EncodeSingle(data, shares[i].Data, i)
		if err != nil {
			b.Fatal(err)
		}
	}

	ret, err := enc.Decode(nil, shares[2:len(shares)-2])
	if err != nil {
		b.Fatal(err)
	}
	return ret
}

func BenchmarkInfecious(b *testing.B) {
	// Create some sample data
	datalen := 64 * 1024
	datanum := 64 * 1024
	data := make([]byte, datalen)
	b.ReportAllocs()
	b.SetBytes(int64(datalen * datanum))

	// Create an encoder with 4 data and 4 parity slices.
	enc, err := infectious.NewFEC(4, 8)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.StopTimer()
	for i := 0; i < datanum; i++ {
		fillRandom(data)
		b.StartTimer()
		buf := benchmarkInfecious(b, enc, data)
		b.StopTimer()
		if !bytes.Equal(buf, data) {
			b.Fatal("recovered bytes do not match")
		}
	}

}
