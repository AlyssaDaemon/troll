package network

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type Response struct {
	URL      string
	Error    error
	Status   int
	Duration time.Duration
}

type Replicator struct {
	Context             context.Context
	Ticker              *time.Ticker
	MaxWorkers          int64
	CurrentWorkers      int64
	SuccessfulCallsMade int64
	ErrorCallsMade      int64
	WorkerSleep         int64
	URLs                []string
	StartTime           time.Time
	StatusStats         map[int]int64
	TimeRunning         time.Duration
	ShortestTime        time.Duration
	LongestTime         time.Duration
}

func (r *Replicator) Stats() {
	totalTime := time.Since(r.StartTime)
	totalCalls := r.SuccessfulCallsMade + r.ErrorCallsMade

	avgRespTime := time.Duration(0)

	if r.SuccessfulCallsMade > 0 {
		avgRespTime = time.Duration(r.TimeRunning.Nanoseconds() / r.SuccessfulCallsMade)
	}

	if r.ShortestTime == time.Duration(9223372036854775807) {
		r.ShortestTime = time.Duration(0)
	}

	fmt.Println(strings.Repeat("\n", 2))
	fmt.Println("Final Stats:")
	fmt.Printf("Total Run Duration: %v\n", totalTime)
	fmt.Printf("Duration spent doing HTTP: %v (May be larger than total duration due to concurrency)\n", r.TimeRunning)
	fmt.Printf("Max Concurrency: %v\n", r.MaxWorkers)
	fmt.Printf("Total Calls: %v (%v successful, %v errors)\n", totalCalls, r.SuccessfulCallsMade, r.ErrorCallsMade)
	fmt.Printf("Avg Response Time: %v\n", avgRespTime)
	fmt.Printf("Shortest Response Time: %v\n", r.ShortestTime)
	fmt.Printf("Longest Response Time %v\n", r.LongestTime)
	if len(r.StatusStats) > 0 {
		fmt.Println("HTTP Code Stats:")
		for i, v := range r.StatusStats {
			fmt.Printf("\t%v:\t%v\n", i, v)
		}
	}

}

func (r *Replicator) Run() {
	r.StartTime = time.Now()
	results := make(chan *Response, r.MaxWorkers)

	for {
		select {
		case <-r.Context.Done():
			return
		case result := <-results:
			if result.Error != nil {
				fmt.Println(result.Error)
				r.ErrorCallsMade++
			} else {
				r.SuccessfulCallsMade++

				fmt.Printf("%v: %v %v\n", result.Status, result.URL, result.Duration)

				if _, ok := r.StatusStats[result.Status]; !ok {
					r.StatusStats[result.Status] = 0
				}
				r.StatusStats[result.Status]++

				r.TimeRunning += result.Duration
				if r.ShortestTime > result.Duration {
					r.ShortestTime = result.Duration
				}

				if r.LongestTime < result.Duration {
					r.LongestTime = result.Duration
				}
			}
			r.CurrentWorkers--
		case <-r.Ticker.C:
			if r.CurrentWorkers >= r.MaxWorkers {
				continue
			}
			for r.MaxWorkers > r.CurrentWorkers {
				go func(url string, sleep int64, done chan<- *Response) {
					if sleep > 0 {
						time.Sleep(time.Duration(rand.Int63n(sleep)) * time.Millisecond)
					}
					startTime := time.Now()
					resp, err := http.Get(url)
					httpDuration := time.Since(startTime)
					status := 0

					if err == nil {
						defer resp.Body.Close()
						status = resp.StatusCode
					}

					response := &Response{
						URL:      url,
						Duration: httpDuration,
						Error:    err,
						Status:   status,
					}

					done <- response

				}(r.URLs[rand.Intn(len(r.URLs))], r.WorkerSleep, results)
				r.CurrentWorkers++
			}
		}
	}
}
