package benchmark

import (
	"context"
	"fmt"
	"math"
	_ "net/http/pprof"
	"testing"
	"time"

	"github.com/Yiling-J/theine-go"
	"github.com/allegro/bigcache/v3"
	"github.com/dgraph-io/ristretto"
	"github.com/leslie-fei/fastcache"
)

const (
	sharding = 128
	capacity = 100
	//benchcount = 1 << 20
	benchcount = 1 << 20
)

var valCount = int(math.Log2(float64(16 * fastcache.MB)))

var (
	benchkeys = make([]string, 0, benchcount)
	benchVals = make([][]byte, 0)
)

func init() {
	for i := 0; i < benchcount; i++ {
		benchkeys = append(benchkeys, fmt.Sprintf("key_%d", i))
	}

	for i := 0; i < valCount; i++ {
		c := i + 1
		benchVals = append(benchVals, make([]byte, int(math.Pow(float64(c), 2))))
	}

}

func getIndex(i int) int {
	return i & (len(benchkeys) - 1)
}

func getValIndex(i int) int {
	return i & (valCount - 1)
}

func BenchmarkFastCache_Set(b *testing.B) {
	cache, err := fastcache.NewCache(fastcache.GB, &fastcache.Config{
		Shards: sharding,
	})
	if err != nil {
		b.Fatal(err)
	}
	var mc = cache
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			value := benchVals[getValIndex(i)]
			mc.Set(benchkeys[index], value)
			i++
		}
	})
}

func BenchmarkFastCache_Get(b *testing.B) {
	cache, err := fastcache.NewCache(fastcache.GB, &fastcache.Config{
		Shards: sharding,
	})
	if err != nil {
		b.Fatal(err)
	}
	mc := cache
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.Set(benchkeys[i%benchcount], value)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			if _, err := mc.Get(benchkeys[index]); err != nil {
				panic(err)
			}
			i++
		}
	})
}

func BenchmarkFastCache_SetAndGet(b *testing.B) {
	cache, err := fastcache.NewCache(fastcache.GB, &fastcache.Config{
		Shards: sharding,
	})
	if err != nil {
		b.Fatal(err)
	}
	mc := cache
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.Set(benchkeys[i%benchcount], value)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			if index&7 == 0 {
				value := benchVals[getValIndex(i)]
				mc.Set(benchkeys[index], value)
			} else {
				mc.Get(benchkeys[index])
			}
			i++
		}
	})
}

func BenchmarkBigCache_Set(b *testing.B) {
	cache, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             sharding,
		LifeWindow:         10 * time.Minute,
		MaxEntriesInWindow: 256,
		MaxEntrySize:       10000,
		Verbose:            false,
	})
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			value := benchVals[getValIndex(i)]
			cache.Set(benchkeys[index], value)
			i++
		}
	})
}

func BenchmarkBigCache_Get(b *testing.B) {
	cache, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             sharding,
		LifeWindow:         10 * time.Minute,
		MaxEntriesInWindow: 256,
		MaxEntrySize:       10000,
		Verbose:            false,
	})
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		cache.Set(benchkeys[i%benchcount], value)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			cache.Get(benchkeys[index])
			i++
		}
	})
}

func BenchmarkBigCache_SetAndGet(b *testing.B) {
	cache, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             sharding,
		LifeWindow:         10 * time.Minute,
		MaxEntriesInWindow: 256,
		MaxEntrySize:       10000,
		Verbose:            false,
	})
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		cache.Set(benchkeys[i%benchcount], value)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			if index&7 == 0 {
				value := benchVals[getValIndex(i)]
				cache.Set(benchkeys[index], value)
			} else {
				cache.Get(benchkeys[index])
			}
			i++
		}
	})
}

func BenchmarkRistretto_Set(b *testing.B) {
	var mc, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: capacity * sharding * 10, // number of keys to track frequency of (10M).
		MaxCost:     1 << 30,                  // maximum cost of cache (1GB).
		BufferItems: 64,                       // number of keys per Get buffer.
	})
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			value := benchVals[getValIndex(i)]
			mc.SetWithTTL(benchkeys[index], value, int64(len(value)), time.Hour)
			i++
		}
	})
}

func BenchmarkRistretto_Get(b *testing.B) {
	var mc, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: capacity * sharding * 10, // number of keys to track frequency of (10M).
		MaxCost:     1 << 30,                  // maximum cost of cache (1GB).
		BufferItems: 64,                       // number of keys per Get buffer.
	})
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.SetWithTTL(benchkeys[i%benchcount], value, int64(len(value)), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			mc.Get(benchkeys[index])
			i++
		}
	})
}

func BenchmarkRistretto_SetAndGet(b *testing.B) {
	var mc, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: capacity * sharding * 10, // number of keys to track frequency of (10M).
		MaxCost:     1 << 30,                  // maximum cost of cache (1GB).
		BufferItems: 64,                       // number of keys per Get buffer.
	})
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.SetWithTTL(benchkeys[i%benchcount], value, int64(len(value)), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			if index&7 == 0 {
				value := benchVals[getValIndex(i)]
				mc.SetWithTTL(benchkeys[index], value, int64(len(value)), time.Hour)
			} else {
				mc.Get(benchkeys[index])
			}
			i++
		}
	})
}

func BenchmarkTheine_Set(b *testing.B) {
	mc, _ := theine.NewBuilder[string, []byte](sharding * capacity).Build()
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			index := getIndex(i)
			value := benchVals[getValIndex(i)]
			i++
			mc.SetWithTTL(benchkeys[index], value, int64(len(value)), time.Hour)
		}
	})
}

func BenchmarkTheine_Get(b *testing.B) {
	mc, _ := theine.NewBuilder[string, []byte](sharding * capacity).Build()
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.SetWithTTL(benchkeys[i%benchcount], value, int64(len(value)), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			index := getIndex(i)
			mc.Get(benchkeys[index])
			i++
		}
	})
}

func BenchmarkTheine_SetAndGet(b *testing.B) {
	mc, _ := theine.NewBuilder[string, []byte](sharding * capacity).Build()
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.SetWithTTL(benchkeys[i%benchcount], value, int64(len(value)), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			index := getIndex(i)
			if index&7 == 0 {
				value := benchVals[getValIndex(i)]
				mc.SetWithTTL(benchkeys[index], value, int64(len(value)), time.Hour)
			} else {
				mc.Get(benchkeys[index])
			}
			i++
		}
	})
}
