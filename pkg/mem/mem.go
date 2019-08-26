package mem

import (
	"context"
	"time"
)

type Replicator struct {
	Ticker             *time.Ticker
	MaxSize            int64
	Context            context.Context
	MaxWorkers         int64
	WorkerQueue        chan error
	CurrentWorkers     int64
	BytesWritten       uint64
	BytesWrittenModulo uint64
	ReleaseMemory      bool
	RAM                []int
}

func (r *Replicator) Run() {

}

func NeverEndingMemoryEater(size int64) {
	for {
		bytes := make([]byte, size)

		for i := range bytes {
			bytes[i] = bytes[i] + 1
		}

	}
}
