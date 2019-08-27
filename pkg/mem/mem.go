package mem

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type Result struct {
	Duration     time.Duration
	BytesWritten int64
	BytesRead    int64
	Error        error
	MagicNumber  int
}

type Replicator struct {
	Ticker         *time.Ticker
	MaxSize        int64
	Context        context.Context
	MaxWorkers     int64
	CurrentWorkers int64
	BytesWritten   uint64
	BytesRead      uint64
	TimeRunning    time.Duration
	ReleaseMemory  bool
	RAM            []int
	Read           bool
	Force          bool
	StartTime      time.Time
	ShortestTime   time.Duration
	LongestTime    time.Duration
	JobsCompleted  int64
}

func (r *Replicator) Stats() {
	totalTime := time.Since(r.StartTime)
	avgRespTime := time.Duration(0)

	if r.JobsCompleted > 0 {
		avgRespTime = time.Duration(r.TimeRunning.Nanoseconds() / r.JobsCompleted)
	}

	if r.ShortestTime == time.Duration(9223372036854775807) {
		r.ShortestTime = time.Duration(0)
	}

	fmt.Println(strings.Repeat("\n", 2))
	fmt.Println("Final Stats:")
	fmt.Printf("Total Run Duration %v\n", totalTime)
	fmt.Printf("Max Concurrency %v\n", r.MaxWorkers)
	fmt.Printf("Jobs Completed: %v\n", r.JobsCompleted)
	fmt.Printf("Bytes Written: %v, Bytes Read: %v\n", r.BytesWritten, r.BytesRead)
	fmt.Printf("Average completion time: %v\n", avgRespTime)
	fmt.Printf("Shortest Job Time %v\n", r.ShortestTime)
	fmt.Printf("Longest Job Time: %v\n", r.LongestTime)

}

func (r *Replicator) Run() {
	r.StartTime = time.Now()
	queue := make(chan *Result, r.MaxWorkers)
	mutex := sync.Mutex{}
	for {
		select {
		case <-r.Context.Done():
			r.Ticker.Stop()
			close(queue)
			return
		case <-r.Ticker.C:

			if r.Force {
				memStatsBeforeGC := runtime.MemStats{}
				runtime.ReadMemStats(&memStatsBeforeGC)
				startTime := time.Now()
				runtime.GC()
				gcTime := time.Since(startTime)
				memStatsAfterGC := runtime.MemStats{}
				runtime.ReadMemStats(&memStatsAfterGC)
				fmt.Printf("GC took %v\n", gcTime)
				fmt.Println("Before GC Stats:")
				fmt.Printf("\tHeapAlloc: %v\n", memStatsBeforeGC.HeapAlloc)
				fmt.Printf("\tSys: %v\n", memStatsBeforeGC.Sys)
				fmt.Printf("\tMallocs: %v\n", memStatsBeforeGC.Mallocs)
				fmt.Printf("\tFrees: %v\n", memStatsBeforeGC.Frees)
				fmt.Printf("\tLive Objects: %v\n", memStatsBeforeGC.Mallocs-memStatsBeforeGC.Frees)
				fmt.Println("After GC Stats: (Difference)")
				fmt.Printf("\tHeapAlloc: %v\n", memStatsAfterGC.HeapAlloc)
				fmt.Printf("\tSys: %v\n", memStatsAfterGC.Sys)
				fmt.Printf("\tLive Objects: %v\n", memStatsAfterGC.Mallocs-memStatsAfterGC.Frees)
				fmt.Printf("\tFrees on GC: %v\n", memStatsAfterGC.Frees-memStatsBeforeGC.Frees)
			}

			if r.CurrentWorkers >= r.MaxWorkers {
				continue
			}

			maxSize := r.MaxSize / r.MaxWorkers

			start := rand.Int63n(maxSize - 1)
			size := rand.Int63n(maxSize - start)

			for i := r.CurrentWorkers; i < r.MaxWorkers; i++ {
				go func(lock *sync.Mutex, size, start int64, release, read bool, results chan<- *Result) {
					magic := 0
					duration := time.Duration(0)
					bytesRead := 0
					bytesWrote := 0
					if !release {
						lock.Lock()
						startTime := time.Now()
						for i := int64(0); i < size; i++ {
							r.RAM[size+i]++
							bytesWrote += int(unsafe.Sizeof(r.RAM[size+i]))
						}
						if read {
							for i := int64(0); i < size; i++ {
								magic += r.RAM[start+i]
								bytesRead += int(unsafe.Sizeof(r.RAM[size+i]))
							}
						}
						duration = time.Since(startTime)
						lock.Unlock()
					} else {
						startTime := time.Now()
						ram := make([]int, 0)

						for i := start; i < size; i++ {
							ram = append(ram, int(i))
							bytesWrote += int(unsafe.Sizeof(ram[i]))
						}
						if read {
							for i := start; i < size; i++ {
								magic += ram[i]
								bytesRead += int(unsafe.Sizeof(ram[i]))
							}
						}
						duration = time.Since(startTime)
					}

					results <- &Result{
						Error:        nil,
						MagicNumber:  magic,
						Duration:     duration,
						BytesWritten: int64(bytesWrote),
						BytesRead:    int64(bytesRead),
					}

				}(&mutex, start, size, r.ReleaseMemory, r.Read, queue)

				r.CurrentWorkers++
			}
		case result := <-queue:
			if result.Error != nil {
				fmt.Printf("Error during Memory Load Test %v\n", result.Error)
				continue
			}

			r.BytesWritten += uint64(result.BytesWritten)
			r.BytesRead += uint64(result.BytesRead)
			r.TimeRunning += result.Duration
			r.JobsCompleted++
			r.CurrentWorkers--

			if r.ShortestTime > result.Duration {
				r.ShortestTime = result.Duration
			}

			if r.LongestTime < result.Duration {
				r.LongestTime = result.Duration
			}

			fmt.Printf("Wrote: %v, Read: %v, MagicNumber: %v, took: %v\n", result.BytesWritten, result.BytesRead, result.MagicNumber, result.Duration)

		}
	}
}

func ParseMemString(memString string) (int64, error) {
	if len(memString) == 0 {
		return 0, fmt.Errorf("String was empty")
	}

	lowerMemString := strings.ToLower(memString)

	mem := int64(0)
	scale := 0
	numIndex := 0

	for i := len(lowerMemString) - 1; i >= 0; i-- {
		char := lowerMemString[i]
		switch char {
		case 'e':
			// scale = exabyte
			scale = 60
		case 'p':
			// scale = petabye
			scale = 50
		case 't':
			// scale = terabyte
			scale = 40
		case 'g':
			// scale = gigabyte
			scale = 30
		case 'm':
			// scale = megabyte
			scale = 20
		case 'k':
			// scale = kilobyte
			scale = 10
		case 'b':
			scale = 0
		default:
			num, err := strconv.ParseInt(string(memString[i]), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("unable to parse %v as an integer %v", string(char), err)
			}
			mem += num * int64(math.Pow10(numIndex))
			numIndex = numIndex + 1
		}
	}

	if scale > 0 {
		mem *= int64(math.Pow(2, float64(scale)))
	}

	return mem, nil
}
