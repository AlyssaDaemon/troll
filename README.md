# Troll

Troll is a load testing application (written for K8s, but can be used locally).

```
Usage: troll <flags> <subcommand> <subcommand args>

Subcommands:
        commands         list all command names
        cpu              Load Test CPU
        files            Load Test Files
        flags            describe all known top-level flags
        help             describe subcommands and their syntax
        mem              Load Test Memory
        network          Load Test Network


Use "troll flags" for a list of top-level flags
```

## CPU
```
cpu [args]:
        Load Test CPU
  -rate int
        How often to display CPU usage (Only used with -show) (default 1000)
  -show
        Should it display CPU Usage while running
  -workers int
        Number of workers to deploy (default 7) // Default is always #CPUs (reported by runtime.NumCPU() - 1)
```
## Network
```
network [args] <urls>:
        Load Test Network
  -file string
        File location to pulls URLs from
  -rate int
        How long a 'tick' is in ms (default 1000)
  -sleep int
        Max number of milliseconds for worker to wait between calls, 0 deactiveates feature (0 is default)
  -workers int
        How many concurrent workers to keep alive (default 1)
```
## Files
```
files [args]:
        Load Test Files
  -bytes
        Write random bytes (instead of the same bytes over and over (default true)
  -every int
        How often should we report writing files? (Set to 0 to disable) (default 1)
  -fill
        Turns on infinitely filling a single file (only works in singleFile mode)
  -path string
        Where should we be writing files to? (default "/tmp")
  -rate int
        How long a 'tick' is in ms (default 1000)
  -single
        Write to a single files instead of multiple
  -size int
        How big should files be? (default 512)
  -workers int
        In multifile, how many files to write per tick (default 1)
```
## Memory
**NOTE: Memory is not yet complete. Still working on this.**
```
mem [args]:
        Load Test Memory
  -force
        Should we force a GC every call?
  -max string
        Max amount in memory in base 2. Supports b,k,m,g,t,p (default "1G")
  -rate int
        How long a 'tick' is in ms (default 1000)
  -read
        Should we test reading memory?
  -release
        Should we release all our memory every tick?
  -workers int
        Max number of workers writing and ready memory (default 1)
```