package shareassembler

import (
	"context"
	"oneway-filesync/pkg/structs"
	"oneway-filesync/pkg/utils"
	"time"
)

// Cache docs:
// For every (FileHash,FileDataOffset) we save a cache of shares
// Since we need at least <required> shares to create the original data we have to cache them somewhere
// After we get <required> shares we can pull them and create the data but then up to (<total>-<required>) will continue coming in
// The LastUpdated is a field which we can time out based upon and
type CacheKey struct {
	Hash       [structs.HASHSIZE]byte
	DataOffset int64
}
type CacheValue struct {
	Shares      chan *structs.Chunk
	LastUpdated time.Time
}

type ShareAssembler struct {
	required int
	total    int
	input    chan *structs.Chunk
	output   chan []*structs.Chunk
	cache    utils.RWMutexMap[CacheKey, *CacheValue]
}

// The manager acts as a "Garbage collector"
// every chunk that didn't get any new shares for the past 60 seconds can be
// assumed to never again receive more shares and deleted
func Manager(ctx context.Context, conf *ShareAssembler) {
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conf.cache.Range(func(key CacheKey, value *CacheValue) bool {
				if time.Since(value.LastUpdated).Seconds() > 60 {
					conf.cache.Delete(key)
				}
				return true
			})
		}
	}
}

func Worker(ctx context.Context, conf *ShareAssembler) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-conf.input:
			value, _ := conf.cache.LoadOrStore(
				CacheKey{Hash: chunk.Hash, DataOffset: chunk.DataOffset},
				&CacheValue{Shares: make(chan *structs.Chunk, conf.total), LastUpdated: time.Now()})
			value.Shares <- chunk
			value.LastUpdated = time.Now()
			conf.cache.Store(CacheKey{Hash: chunk.Hash, DataOffset: chunk.DataOffset}, value)
			if len(value.Shares) >= conf.required {
				value, loaded := conf.cache.LoadAndDelete(CacheKey{Hash: chunk.Hash, DataOffset: chunk.DataOffset})
				if loaded && (len(value.Shares) >= conf.required) {
					n := len(value.Shares)
					var shares []*structs.Chunk
					for i := 0; i < n; i++ {
						shares = append(shares, <-value.Shares)
					}
					conf.output <- shares
				}
			}
		}
	}
}

func CreateShareAssembler(ctx context.Context, required int, total int, input chan *structs.Chunk, output chan []*structs.Chunk, workercount int) {
	conf := ShareAssembler{
		required: required,
		total:    total,
		input:    input,
		output:   output,
		cache:    utils.RWMutexMap[CacheKey, *CacheValue]{},
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, &conf)
	}
	go Manager(ctx, &conf)
}
