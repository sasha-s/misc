package pump

import (
	"runtime"
	"sync"
	"testing"

	lfc "github.com/PurpureGecko/go-lfc"
)

const n = 10

var blockSize = 1024 * 16
var numBlocks = 128 / 4

func BenchmarkPump(b *testing.B) {
	p := New(blockSize, numBlocks)
	arr := make([]int, blockSize*numBlocks)
	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < b.N/blockSize; k++ {
				b := p.StartWrite()
				for u := b.Start; u < b.End; u++ {
					arr[u]++
				}
				p.CommitWrite(b, b.End-b.Start)
			}
		}()
		wg.Add(1)
		go func() {
			sum := 0
			defer wg.Done()
			for k := 0; k < b.N/blockSize; k++ {
				b := p.StartRead()
				for u := b.Start; u < b.End; u++ {
					sum += arr[u]
				}
				p.CommitRead(b)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkChan(b *testing.B) {
	ch := make(chan int, blockSize*numBlocks)
	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < b.N; k++ {
				ch <- k
			}
		}()
		wg.Add(1)
		go func() {
			sum := 0
			defer wg.Done()
			for k := 0; k < b.N; k++ {
				sum += <-ch
			}
		}()
	}
	wg.Wait()
}

func BenchmarkQ(b *testing.B) {
	q := lfc.NewQueue()
	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < b.N; k++ {
				q.Enqueue(k)
			}
		}()
		wg.Add(1)
		go func() {
			sum := 0
			defer wg.Done()
			for k := 0; k < b.N; k++ {
				for {
					v, ok := q.Dequeue()
					if !ok {
						runtime.Gosched()
						continue
					}
					sum += v.(int)
					break
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkMu(b *testing.B) {
	arr := make([]int, blockSize*numBlocks)
	var mu sync.Mutex
	var size int
	var pos int
	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < b.N; k++ {
				for {
					mu.Lock()
					if size == len(arr) {
						mu.Unlock()
						runtime.Gosched()
						continue
					}
					break
				}
				arr[pos]++
				pos++
				size++
				if pos == len(arr) {
					pos = 0
				}
				mu.Unlock()
			}
		}()
		wg.Add(1)
		go func() {
			sum := 0
			defer wg.Done()
			for k := 0; k < b.N; k++ {
				for {
					mu.Lock()
					if size == 0 {
						mu.Unlock()
						runtime.Gosched()
						continue
					}
					break
				}
				sum += arr[(pos+size)%len(arr)]
				size--
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
}
