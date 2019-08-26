package cpu

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var cpuRegex = regexp.MustCompile("^cpu[0-9]*")

type CPUStat struct {
	Idle  uint64
	Total uint64
}

type CPUSample struct {
	CPUStats  []*CPUStat
	Timestamp time.Time
}

type Replicator struct {
	Ticker     *time.Ticker
	Context    context.Context
	Workers    int64
	DisplayCPU bool
	lastSample *CPUSample
}

func (r *Replicator) Run() {

	for i := int64(0); i < r.Workers; i++ {
		go func() {
			for {
			}
		}()
	}

	for {
		select {
		case <-r.Context.Done():
			return
		case <-r.Ticker.C:
			if r.DisplayCPU {
				sample, err := getCPUSample()

				if err != nil {
					fmt.Printf("Error getting CPU Samples: %v\n", err)
					continue
				}

				if r.lastSample != nil {
					percentages, err := getCPUPercentage([]*CPUSample{r.lastSample, sample})

					if err != nil {
						fmt.Printf("Error getting CPU Percentage due to %v\n", err)
						continue
					}

					log := strings.Builder{}
					log.WriteString("CPU Report:\n")

					for i, percentage := range percentages {
						log.WriteString(fmt.Sprintf("\tCPU %v: %v%%\n", i, percentage))
					}

					fmt.Println(log.String())
				}

				r.lastSample = sample

			}
		}
	}
}

func getCPUSample() (*CPUSample, error) {
	sample := CPUSample{
		CPUStats:  make([]*CPUStat, runtime.NumCPU()),
		Timestamp: time.Now(),
	}

	contents, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return nil, err
	}
	total := uint64(0)
	idle := uint64(0)
	lines := strings.Split(string(contents), "\n")
	for i, line := range lines {
		fields := strings.Fields(line)
		if cpuRegex.MatchString(fields[0]) {
			for i, field := range fields {
				val, err := strconv.ParseUint(field, 10, 64)
				if err != nil {
					fmt.Println("Error: ", i, field, err)
					return nil, err
				}
				total += val // tally up all the numbers to get total ticks
				if i == 4 {  // idle is the 5th field in the cpu line
					idle = val
				}
			}
			sample.CPUStats[i] = &CPUStat{
				Idle:  idle,
				Total: total,
			}
		}
	}
	return &sample, fmt.Errorf("unable to parse /proc/stat")
}

func getCPUPercentage(samples []*CPUSample) ([]float64, error) {
	cpus := make([]float64, runtime.NumCPU())

	if len(samples) < 2 {
		return cpus, fmt.Errorf("got too few samples %v", len(samples))
	}

	latest := samples[len(samples)]
	previous := samples[len(samples)-1]

	for i := range cpus {
		idleTicks := float64(latest.CPUStats[i].Idle - previous.CPUStats[i].Idle)
		totalTicks := float64(latest.CPUStats[i].Total - previous.CPUStats[i].Total)

		cpus[i] = 100 * (totalTicks - idleTicks) / totalTicks
	}

	return cpus, nil
}
