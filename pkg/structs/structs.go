package structs

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"oneway-filesync/pkg/zip"
	"os"
	"time"

	"github.com/zhuangsirui/binpacker"
)

const HASHSIZE = 32 // Using the sha256.Size as const directly causes linting issues

func HashFile(f *os.File, encrypted bool) ([HASHSIZE]byte, error) {
	var ret [HASHSIZE]byte
	h := sha256.New()

	var err error
	if encrypted {
		err = zip.ZipFile(h, f)
	} else {
		_, err = io.Copy(h, f)
	}
	if err != nil {
		return ret, err
	}

	hash := h.Sum(nil)
	copy(ret[:], hash)
	return ret, nil
}

type Chunk struct {
	Path        string
	Hash        [HASHSIZE]byte
	Encrypted   bool
	DataOffset  int64
	DataPadding uint32
	ShareIndex  uint32
	Data        []byte
}

var b2i = map[bool]byte{false: 0, true: 1}

var i2b = [2]bool{false, true}

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
	packer.PushByte(b2i[c.Encrypted])
	packer.PushInt64(c.DataOffset)
	packer.PushUint32(c.DataPadding)
	packer.PushUint32(c.ShareIndex)
	packer.PushUint32(uint32(len(c.Data)))
	packer.PushBytes(c.Data)

	return buffer.Bytes(), packer.Error()
}

// Decode binary buffer into a Chunk object
func DecodeChunk(data []byte) (Chunk, error) {
	var c Chunk

	buffer := bytes.NewBuffer(data)
	unpacker := binpacker.NewUnpacker(binary.BigEndian, buffer)
	unpacker.StringWithUint32Prefix(&c.Path)
	var hashslice []byte
	unpacker.FetchBytes(uint64(HASHSIZE), &hashslice)
	copy(c.Hash[:], hashslice)
	var enc byte
	unpacker.FetchByte(&enc)
	c.Encrypted = i2b[enc]
	unpacker.FetchInt64(&c.DataOffset)
	unpacker.FetchUint32(&c.DataPadding)
	unpacker.FetchUint32(&c.ShareIndex)
	unpacker.BytesWithUint32Prefix(&c.Data)

	return c, unpacker.Error()
}

type OpenTempFile struct {
	TempFile    string
	Path        string
	Hash        [HASHSIZE]byte
	Encrypted   bool
	LastUpdated time.Time
}
