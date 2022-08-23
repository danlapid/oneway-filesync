package shareassembler

import (
	"context"
	"oneway-filesync/pkg/structs"
	"sync"
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
	Shares      chan structs.Chunk
	LastUpdated time.Time
}

type ShareAssembler struct {
	required int
	total    int
	input    chan structs.Chunk
	output   chan []structs.Chunk
	cache    sync.Map
}

// The manager acts as a "Garbage collector"
// every chunk that didn't get any new shares for the past 60 seconds can be
// assumed to never again receive more shares and deleted
func Manager(ctx context.Context, conf ShareAssembler) {
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conf.cache.Range(func(k, v interface{}) bool {
				key, _ := k.(CacheKey)
				value, _ := v.(CacheValue)
				if time.Since(value.LastUpdated) > 60*time.Second {
					conf.cache.Delete(key)
				}
				return true
			})
		}
	}
}

func Worker(ctx context.Context, conf ShareAssembler) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-conf.input:
			v, _ := conf.cache.LoadOrStore(
				CacheKey{Hash: chunk.Hash, DataOffset: chunk.DataOffset},
				CacheValue{Shares: make(chan structs.Chunk, conf.total), LastUpdated: time.Now()})
			value, _ := v.(CacheValue)
			value.Shares <- chunk
			value.LastUpdated = time.Now()
			conf.cache.Store(CacheKey{Hash: chunk.Hash, DataOffset: chunk.DataOffset}, value)

			if len(value.Shares) >= conf.required {
				v, loaded := conf.cache.LoadAndDelete(CacheKey{Hash: chunk.Hash, DataOffset: chunk.DataOffset})
				if loaded {
					value, _ = v.(CacheValue)

					shares := make([]structs.Chunk, len(value.Shares))
					for i := range value.Shares {
						shares = append(shares, i)
					}

					conf.output <- shares
				}
			}
		}
	}
}

func CreateShareAssembler(ctx context.Context, required int, total int, input chan structs.Chunk, output chan []structs.Chunk, workercount int) {
	conf := ShareAssembler{
		required: required,
		total:    total,
		input:    input,
		output:   output,
		cache:    sync.Map{},
	}
	for i := 0; i < workercount; i++ {
		go Worker(ctx, conf)
	}
	go Manager(ctx, conf)
}
