package files

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Response struct {
	Path     string
	Error    error
	Written  int64
	Duration time.Duration
}

type Replicator struct {
	RootPath           string
	Ticker             *time.Ticker
	MaxSize            int64
	Context            context.Context
	RandomBytes        bool
	MaxWorkers         int64
	CurrentWorkers     int64
	FilesWritten       int64
	FilesWrittenModulo int64
	StartTime          time.Time
	ErrorFiles         int64
	TimeRunning        time.Duration
	ShortestTime       time.Duration
	LongestTime        time.Duration
	TotalBytes         int64
}

func (r *Replicator) Stats() {
	totalTime := time.Since(r.StartTime)
	totalFiles := r.FilesWritten + r.ErrorFiles

	avgRespTime := time.Duration(0)

	if r.SucessfulFiles > 0 {
		avgRespTime = time.Duration(r.TimeRunning.Nanoseconds() / r.SucessfulFiles)
	}

	if r.ShortestTime == time.Duration(9223372036854775807) {
		r.ShortestTime = time.Duration(0)
	}

	fmt.Println(strings.Repeat("\n", 2))
	fmt.Println("Final Stats:")
	fmt.Printf("Total Run Duration %v\n", totalTime)
	fmt.Printf("Max Concurrency %v\n", r.MaxWorkers)
	fmt.Printf("Bytes Written %v (total) %v\n", r.MaxSize, r.MaxSize*r.FilesWritten)
	fmt.Printf("Total calls: %v (%v sucessfull, %v errors)\n", totalFiles, r.FilesWritten, r.ErrorFiles)
	fmt.Printf("Avg Write Time: %v\n", avgRespTime)
	fmt.Printf("Shortest Write Time: %v\n", r.ShortestTime)
	fmt.Printf("Longest Response Time %v\n", r.LongestTime)

}

func (r *Replicator) Run() {
	r.StartTime = time.Now()
	results := make(chan *Response, r.MaxWorkers)
	bytes := make([]byte, r.MaxSize)

	if !r.RandomBytes {
		_, err := rand.Read(bytes)

		if err != nil {
			fmt.Println(err)
		}
	}

	for {
		select {
		case <-r.Context.Done():
			return
		case result := <-results:
			if result.Error != nil {
				fmt.Printf("Got an error %v\n", result)
				r.ErrorFiles++
			} else {
				r.FilesWritten++
				fmt.Printf("%v: %v bytes %v\n", result.Path, result.Written, result.Duration)

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
				fileName, err := uuid.NewRandom()

				if err != nil {
					fmt.Printf("Error creating UUID due to %v\n", err)
					continue
				}

				path := path.Join(r.RootPath, fileName.String())
				size := rand.Int63n(r.MaxSize)

				if !r.RandomBytes {
					go wrappedWriteFile(path, bytes, results)
				} else {
					go wrappedCreateAndWriteFile(path, size, results)
				}

				r.CurrentWorkers++
			}

		}
	}
}

func wrappedCreateAndWriteFile(path string, size int64, pipe chan<- *Response) {
	startTime := time.Now()
	written, err := CreateAndWriteFile(path, size)
	duration := time.Since(startTime)

	pipe <- &Response{
		Path:     path,
		Error:    err,
		Written:  int64(written),
		Duration: duration,
	}
}

func wrappedWriteFile(path string, body []byte, pipe chan<- *Response) {
	startTime := time.Now()
	written, err := writeFile(path, body)
	duration := time.Since(startTime)

	pipe <- &Response{
		Path:     path,
		Error:    err,
		Written:  int64(written),
		Duration: duration,
	}
}

func writeFile(path string, body []byte) (int, error) {
	file, err := os.Create(path)

	if err != nil {
		return 0, err
	}

	defer file.Close()

	return file.Write(body)
}

// CreateAndWriteFile creates and writes a file with a size of size filled with random bytes
func CreateAndWriteFile(path string, size int64) (int, error) {
	bytes := make([]byte, size)

	_, err := rand.Read(bytes)

	if err != nil {
		return 0, err
	}

	return writeFile(path, bytes)

}

func NeverEndingRandomFile(path string, size int64) error {
	bytes := make([]byte, size)
	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	for {
		_, err := rand.Read(bytes)

		if err != nil {
			return err
		}

		_, err = file.Write(bytes)

		if err != nil {
			return err
		}

		err = file.Sync()

		if err != nil {
			return err
		}

	}
}

func NeverEndingFile(path string, size int64) error {
	bytes := make([]byte, size)
	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	for {

		_, err = file.Write(bytes)

		if err != nil {
			return err
		}

	}
}
