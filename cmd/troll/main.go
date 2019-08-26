package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/google/subcommands"

	"github.com/alyssadaemon/troll/pkg/cpu"
	"github.com/alyssadaemon/troll/pkg/files"
	"github.com/alyssadaemon/troll/pkg/mem"
	"github.com/alyssadaemon/troll/pkg/network"
)

var httpRegex = regexp.MustCompile("^https?://")

type FilesCommand struct {
	rootPath        string
	maxSize         int64
	singleFile      bool
	randomBytes     bool
	fillFile        bool
	maxWorkers      int64
	replicationRate int64
}

func (*FilesCommand) Name() string {
	return "files"
}

func (*FilesCommand) Synopsis() string {
	return "Load Test Files"
}

func (f *FilesCommand) Usage() string {
	usage := strings.Builder{}
	usage.WriteString(fmt.Sprintf("%s [args]:\n", f.Name()))
	usage.WriteString(fmt.Sprintf("\t%s\n", f.Synopsis()))

	return usage.String()

}

func (f *FilesCommand) SetFlags(flags *flag.FlagSet) {
	flags.BoolVar(&f.singleFile, "single", false, "Write to a single files instead of multiple")
	flags.BoolVar(&f.randomBytes, "bytes", true, "Write random bytes (instead of the same bytes over and over")
	flags.BoolVar(&f.fillFile, "fill", false, "Turns on infinitely filling a single file (only works in singleFile mode)")
	flags.StringVar(&f.rootPath, "path", "/tmp", "Where should we be writing files to?")
	flags.Int64Var(&f.maxSize, "size", 512, "How big should files be?")
	flags.Int64Var(&f.maxWorkers, "workers", 1, "In multifile, how many files to write per tick")
	flags.Int64Var(&f.replicationRate, "rate", 1000, "How long a 'tick' is in ms")
}

func (f *FilesCommand) Execute(ctx context.Context, flags *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	if f.singleFile {

		if f.fillFile {

			if f.randomBytes {
				err := files.NeverEndingRandomFile(f.rootPath, f.maxSize)
				if err != nil {
					fmt.Println(err)
					return subcommands.ExitFailure
				}
			} else {
				err := files.NeverEndingFile(f.rootPath, f.maxSize)
				if err != nil {
					fmt.Println(err)
					return subcommands.ExitFailure
				}
			}

		} else {
			_, err := files.CreateAndWriteFile(f.rootPath, f.maxSize)
			if err != nil {
				fmt.Println(err)
				return subcommands.ExitFailure
			}

		}

	} else {
		replicator := files.Replicator{
			RootPath:     f.rootPath,
			Ticker:       time.NewTicker(time.Duration(f.replicationRate) * time.Millisecond),
			MaxSize:      f.maxSize,
			MaxWorkers:   f.maxWorkers,
			Context:      ctx,
			RandomBytes:  f.randomBytes,
			ShortestTime: time.Duration(9223372036854775807),
		}

		replicator.Run()
		replicator.Stats()
	}

	return subcommands.ExitSuccess
}

type NetworkCommand struct {
	URLFile         string
	replicationRate int64
	maxWorkers      int64
	workerSleep     int64
}

func (*NetworkCommand) Name() string {
	return "network"
}

func (*NetworkCommand) Synopsis() string {
	return "Load Test Network"
}

func (n *NetworkCommand) Usage() string {
	usage := strings.Builder{}

	usage.WriteString(fmt.Sprintf("%s [args] <urls>:\n", n.Name()))
	usage.WriteString(fmt.Sprintf("\t%s\n", n.Synopsis()))

	return usage.String()
}

func (n *NetworkCommand) SetFlags(flags *flag.FlagSet) {
	flags.StringVar(&n.URLFile, "file", "", "File location to pulls URLs from")
	flags.Int64Var(&n.replicationRate, "rate", 1000, "How long a 'tick' is in ms")
	flags.Int64Var(&n.maxWorkers, "workers", 1, "How many concurrent workers to keep alive")
	flags.Int64Var(&n.workerSleep, "sleep", 0, "Max number of milliseconds for worker to wait between calls, 0 deactiveates feature (0 is default)")
}

func (n *NetworkCommand) Execute(ctx context.Context, flags *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	args := flags.Args()
	urls := []string{}

	if len(args) > 0 {
		// This is really ugly, but ensures we can be flexible if
		// it's sent as comma seperated or space seperated
		urls = strings.Split(strings.Join(args, ","), ",")
	}

	if n.URLFile != "" {
		output, err := ioutil.ReadFile(n.URLFile)

		if err != nil {
			fmt.Println(err)
			return subcommands.ExitFailure
		}

		lines := strings.Split(string(output), "\n")

		validUrls := make([]string, 0)

		for _, url := range lines {

			trimmedURL := strings.TrimSpace(url)

			if len(trimmedURL) == 0 || []rune(trimmedURL)[0] == '#' || !httpRegex.MatchString(trimmedURL) {
				continue
			}

			validUrls = append(validUrls, trimmedURL)

		}

		urls = validUrls

	}

	if len(urls) == 0 {
		fmt.Println(fmt.Errorf("Empty urls list, unable to continue %v", urls))
		return subcommands.ExitFailure
	}

	replicator := network.Replicator{
		Context:      ctx,
		Ticker:       time.NewTicker(time.Duration(n.replicationRate) * time.Millisecond),
		MaxWorkers:   n.maxWorkers,
		WorkerSleep:  n.workerSleep,
		StatusStats:  make(map[int]int64),
		URLs:         urls,
		ShortestTime: time.Duration(9223372036854775807),
	}

	replicator.Run()
	replicator.Stats()

	return subcommands.ExitSuccess
}

type CPUCommand struct {
	Workers     int64
	DisplayCPU  bool
	DisplayRate int64
}

func (*CPUCommand) Name() string {
	return "cpu"
}

func (*CPUCommand) Synopsis() string {
	return "Load Test CPU"
}

func (c *CPUCommand) Usage() string {
	usage := strings.Builder{}
	usage.WriteString(fmt.Sprintf("%v [args]:\n", c.Name()))
	usage.WriteString(fmt.Sprintf("\t%s\n", c.Synopsis()))
	return usage.String()
}

func (c *CPUCommand) SetFlags(flags *flag.FlagSet) {
	cpus := int64(runtime.NumCPU())

	if cpus > 1 {
		cpus--
	}

	flags.Int64Var(&c.Workers, "workers", cpus, "Number of workers to deploy")
	flags.BoolVar(&c.DisplayCPU, "show", false, "Should it display CPU Usage while running")
	flags.Int64Var(&c.DisplayRate, "rate", 1000, "How often to display CPU usage (Only used with -show)")
}

func (c *CPUCommand) Execute(ctx context.Context, flags *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	replicator := cpu.Replicator{
		Context:    ctx,
		Ticker:     time.NewTicker(time.Duration(c.DisplayRate) * time.Millisecond),
		Workers:    c.Workers,
		DisplayCPU: c.DisplayCPU,
	}

	replicator.Run()

	return subcommands.ExitSuccess
}

type MemoryCommand struct {
	replicationRate int64
}

func (*MemoryCommand) Name() string {
	return "mem"
}

func (*MemoryCommand) Synopsis() string {
	return "Load Test Memory"
}

func (m *MemoryCommand) Usage() string {
	usage := strings.Builder{}
	usage.WriteString(fmt.Sprintf("%v [args]:\n", m.Name()))
	usage.WriteString(fmt.Sprintf("\t%s\n", m.Synopsis()))
	return usage.String()
}

func (m *MemoryCommand) SetFlags(flags *flag.FlagSet) {
	flags.Int64Var(&m.replicationRate, "rate", 1000, "How long a 'tick' is in ms")

}

func (m *MemoryCommand) Execute(ctx context.Context, flags *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	replicator := mem.Replicator{
		Ticker: time.NewTicker(time.Duration(m.replicationRate) * time.Millisecond),
	}

	replicator.Run()

	return subcommands.ExitSuccess

}

func main() {
	done := make(chan os.Signal, 1)

	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&FilesCommand{}, "")
	subcommands.Register(&NetworkCommand{}, "")
	subcommands.Register(&CPUCommand{}, "")
	subcommands.Register(&MemoryCommand{}, "")

	flag.Parse()
	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		os.Exit(int(subcommands.Execute(ctx)))
	}()

	// for loop is so the cancelFunc has time to clean up.
	// we don't actually want to exit until the gofunc calls "os.Exit"
	// Doesn't eat any CPU while waiting for a signal
	for {
		sig := <-done
		fmt.Printf("Got a %v signal, existing!\n", sig)
		cancelFunc()
	}

}
