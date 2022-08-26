package shareassembler

import (
	"context"
	"oneway-filesync/pkg/structs"
	"oneway-filesync/pkg/utils"
	"sync"
	"sync/atomic"
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
	LastUpdated atomic.Int64
	Lock        sync.Mutex
}

type ShareAssembler struct {
	required int
	total    int
	input    chan *structs.Chunk
	output   chan []*structs.Chunk
	cache    utils.RWMutexMap[CacheKey, *CacheValue]
}

// The manager acts as a "Garbage collector"
// every chunk that didn't get any new shares for the past 10 seconds can be
// assumed to never again receive more shares and deleted
func manager(ctx context.Context, conf *ShareAssembler) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conf.cache.Range(func(key CacheKey, value *CacheValue) bool {
				lastUpdated := value.LastUpdated.Load()
				if lastUpdated != 0 && (time.Now().Unix()-lastUpdated) > 10 {
					conf.cache.Delete(key)
				}
				return true
			})
		}
	}
}

func worker(ctx context.Context, conf *ShareAssembler) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-conf.input:
			value, _ := conf.cache.LoadOrStore(
				CacheKey{Hash: chunk.Hash, DataOffset: chunk.DataOffset},
				&CacheValue{Shares: make(chan *structs.Chunk, conf.total*2)})
			value.Shares <- chunk
			value.LastUpdated.Store(time.Now().Unix())

			aquired := value.Lock.TryLock()
			if aquired {
				if len(value.Shares) >= conf.required {
					var shares []*structs.Chunk
					for i := 0; i < conf.required; i++ {
						shares = append(shares, <-value.Shares)
					}
					value.Lock.Unlock()
					conf.output <- shares
				} else {
					value.Lock.Unlock()
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
		go worker(ctx, &conf)
	}
	go manager(ctx, &conf)
}
