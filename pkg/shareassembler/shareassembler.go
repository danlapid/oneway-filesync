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
type cacheKey struct {
	hash       [structs.HASHSIZE]byte
	dataOffset int64
}
type cacheValue struct {
	shares      chan *structs.Chunk
	lastUpdated atomic.Int64
	lock        sync.Mutex
}

type shareAssemblerConfig struct {
	required int
	total    int
	input    chan *structs.Chunk
	output   chan []*structs.Chunk
	cache    utils.RWMutexMap[cacheKey, *cacheValue]
}

// The manager acts as a "Garbage collector"
// every chunk that didn't get any new shares for the past 10 seconds can be
// assumed to never again receive more shares and deleted
func manager(ctx context.Context, conf *shareAssemblerConfig) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conf.cache.Range(func(key cacheKey, value *cacheValue) bool {
				lastUpdated := value.lastUpdated.Load()
				if lastUpdated != 0 && (time.Now().Unix()-lastUpdated) > 10 {
					conf.cache.Delete(key)
				}
				return true
			})
		}
	}
}

func worker(ctx context.Context, conf *shareAssemblerConfig) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-conf.input:
			value, _ := conf.cache.LoadOrStore(
				cacheKey{hash: chunk.Hash, dataOffset: chunk.DataOffset},
				&cacheValue{shares: make(chan *structs.Chunk, conf.total*2)})
			value.shares <- chunk
			value.lastUpdated.Store(time.Now().Unix())

			aquired := value.lock.TryLock()
			if aquired {
				if len(value.shares) >= conf.required {
					var shares []*structs.Chunk
					for i := 0; i < conf.required; i++ {
						shares = append(shares, <-value.shares)
					}
					value.lock.Unlock()
					conf.output <- shares
				} else {
					value.lock.Unlock()
				}
			}
		}
	}
}

func CreateShareAssembler(ctx context.Context, required int, total int, input chan *structs.Chunk, output chan []*structs.Chunk, workercount int) {
	conf := shareAssemblerConfig{
		required: required,
		total:    total,
		input:    input,
		output:   output,
		cache:    utils.RWMutexMap[cacheKey, *cacheValue]{},
	}
	for i := 0; i < workercount; i++ {
		go worker(ctx, &conf)
	}
	go manager(ctx, &conf)
}
