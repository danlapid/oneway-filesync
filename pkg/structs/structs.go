package structs

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/zhuangsirui/binpacker"
)

const HASHSIZE int = sha256.Size

var HashNew = sha256.New

type Chunk struct {
	Path       string
	Hash       [HASHSIZE]byte
	DataIndex  uint32
	ShareIndex uint32
	Data       []byte
}

// Returns the percise overhead of a chunk
// Dependant on path since it's the only variable-length field in a chunk
// This value is required to ensure that every network chunk is of the configured size
func ChunkOverhead(path string) int {
	enc, _ := Chunk{Path: path}.Encode()
	return len(enc)
}

// Encode chunk into binary buffer
// No extravagant serialization library was used in order to be 100% what the overhead will be
func (c Chunk) Encode() ([]byte, error) {
	pathbytes := []byte(c.Path)

	buffer := new(bytes.Buffer)
	packer := binpacker.NewPacker(binary.BigEndian, buffer)
	packer.PushUint32(uint32(len(pathbytes)))
	packer.PushBytes(pathbytes)
	packer.PushBytes(c.Hash[:])
	packer.PushUint32(c.DataIndex)
	packer.PushUint32(c.ShareIndex)
	packer.PushUint32(uint32(len(c.Data)))
	packer.PushBytes(c.Data)

	return buffer.Bytes(), packer.Error()
}

// Decode binary buffer into a Chunk object
func DecodeChunk(data []byte) error {
	var c Chunk

	buffer := bytes.NewBuffer(data)
	unpacker := binpacker.NewUnpacker(binary.BigEndian, buffer)
	unpacker.StringWithUint32Prefix(&c.Path)
	var hashslice []byte
	unpacker.FetchBytes(uint64(HASHSIZE), &hashslice)
	copy(c.Hash[:], hashslice)
	unpacker.FetchUint32(&c.DataIndex)
	unpacker.FetchUint32(&c.ShareIndex)
	unpacker.BytesWithUint32Prefix(&c.Data)

	return unpacker.Error()
}
