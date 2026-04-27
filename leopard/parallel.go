package leopard

import (
	"runtime"
	"sync"
)

// parallelThreshold: minimum number of independent butterfly groups to parallelize.
const parallelThreshold = 4

var workerPool = runtime.NumCPU()

func init() {
	if workerPool < 1 {
		workerPool = 1
	}
}

// parallelFor runs fn(start, end) in parallel across [0, total).
func parallelFor(total int, fn func(start, end int)) {
	nWorkers := workerPool
	if total < nWorkers*parallelThreshold {
		fn(0, total)
		return
	}
	if nWorkers > total {
		nWorkers = total
	}

	var wg sync.WaitGroup
	perWorker := (total + nWorkers - 1) / nWorkers
	for w := 0; w < nWorkers; w++ {
		start := w * perWorker
		end := start + perWorker
		if end > total {
			end = total
		}
		if start >= end {
			break
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			fn(s, e)
		}(start, end)
	}
	wg.Wait()
}

// parallelXorSlices XORs work[i] ^= other[i] for i in [0, count) in parallel.
func parallelXorSlices(work, other [][]byte, count int) {
	if count < parallelThreshold*workerPool {
		for i := 0; i < count; i++ {
			xorSlice(work[i], other[i])
		}
		return
	}
	parallelFor(count, func(start, end int) {
		for i := start; i < end; i++ {
			xorSlice(work[i], other[i])
		}
	})
}
