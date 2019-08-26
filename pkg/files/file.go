package files

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
)

type Replicator struct {
	RootPath           string
	Ticker             *time.Ticker
	MaxSize            int64
	Context            context.Context
	RandomBytes        bool
	MaxWorkers         int64
	CurrentWorkers     int64
	FilesWritten       uint64
	FilesWrittenModulo uint64
}

func (r *Replicator) Run() {
	results := make(chan error, r.MaxWorkers)
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
			if result != nil {
				fmt.Printf("Got an error %v\n", result)
			} else {
				r.FilesWritten++

				if r.FilesWritten%r.FilesWrittenModulo == 0 {
					fmt.Printf("Files written: %v\n", r.FilesWritten)
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

func wrappedCreateAndWriteFile(path string, size int64, pipe chan<- error) {
	pipe <- CreateAndWriteFile(path, size)
}

func wrappedWriteFile(path string, body []byte, pipe chan<- error) {
	pipe <- writeFile(path, body)
}

func writeFile(path string, body []byte) error {
	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.Write(body)

	if err != nil {
		return err
	}

	return nil
}

// CreateAndWriteFile creates and writes a file with a size of size filled with random bytes
func CreateAndWriteFile(path string, size int64) error {
	bytes := make([]byte, size)

	_, err := rand.Read(bytes)

	if err != nil {
		return err
	}

	file, err := os.Create(path)

	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(bytes)

	if err != nil {
		return err
	}

	return nil
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
