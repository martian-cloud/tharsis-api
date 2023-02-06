package jobexecutor

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/mitchellh/go-ps"
	"github.com/prometheus/procfs"
)

const (

	// Path to /proc in case it exists.
	procPath = "/proc"

	// Terraform CLI elapsed times have been seen as short as 50-100 milliseconds (for a tiny module).

	// CLI process monitor interval while waiting for a child process or measuring one.
	monitorInterval = 100 * time.Millisecond

	// Fractions of memory limit to trigger warnings.
	memoryLimitWarningFraction1 = 0.8
	memoryLimitWarningFraction2 = 0.9
)

// MemoryMonitor implements all memory monitor functions.
// The methods should be called in the order shown here.
// GetMaxMemoryUsage can be called multiple times if desired.
type MemoryMonitor interface {
	Start(ctx context.Context)
	Stop()
}

type memoryMonitor struct {
	doneChannel chan bool
	jobLogger   *jobLogger
	memoryLimit uint64
}

// NewMemoryMonitor creates an instance of the memory monitor.
func NewMemoryMonitor(jobLogger *jobLogger, memoryLimit uint64) (MemoryMonitor, error) {

	// error out if the limit is zero.
	if memoryLimit == 0 {
		return nil, fmt.Errorf("memory limit must not be zero")
	}

	return &memoryMonitor{
		doneChannel: make(chan bool),
		memoryLimit: memoryLimit,
		jobLogger:   jobLogger,
	}, nil
}

func (mm *memoryMonitor) Start(ctx context.Context) {
	if checkProc() {
		go func() {
			mm.monitorProcesses()
		}()
	}
}

func (mm *memoryMonitor) Stop() {
	close(mm.doneChannel)
}

// checkProc return true iff there is a directory called /proc.
func checkProc() bool {
	stats, err := os.Stat(procPath)
	if err != nil {
		// If it doesn't exist or cannot be checked, it won't work.
		return false
	}
	return stats.IsDir()
}

// monitorProcesses runs in the parallel Goroutine.
// Because there might be multiple descendant processes and they might evolve over time,
// the main loop must look for new processes each pass.
func (mm *memoryMonitor) monitorProcesses() {

	// Initialize variables used in the loop.  Start with this process.
	// Eligible parents include this process and eventually all its descendants.
	eligibleParents := map[int]struct{}{
		os.Getpid(): {},
	}

	// Because the loop will check for bytes many times during a job,
	// convert threshold percentages to bytes ahead of time.
	warnMemoryUseAt1 := mm.limitFractionToBytes(memoryLimitWarningFraction1)
	warnMemoryUseAt2 := mm.limitFractionToBytes(memoryLimitWarningFraction2)

	// Look first for the lower threshold.
	warnMemoryUseAt := warnMemoryUseAt1

	// Main loop: keep going until the main line signals that Terraform execution has finished.
	for {

		// Sleep a bit up front, because doing the sleep here allows a continue later to not go 100% CPU-bound.
		// It's going to take a while for the Terraform CLI process to get started, so we're not losing anything.
		time.Sleep(monitorInterval)

		// Get a list of all processes currently on the system.
		allProcs, err := ps.Processes()
		if err != nil {
			mm.logError(err)
			// Don't log the same error many times over.
			break
		}

		// Scan for any new descendants.
		// Note: It is understood that this loop will pick up only one new generation of descendants
		// per scan.  The Terraform CLI is not expected to create a deep tree of descendants, and a
		// job should take more than a couple of seconds, so that is expected to be okay.
		allProcsMap := make(map[int]struct{})
		for _, candidate := range allProcs {
			allProcsMap[candidate.Pid()] = struct{}{}

			// Check the candidate's parent PID.
			if _, ok := eligibleParents[candidate.PPid()]; ok {
				// Check whether we have already seen it.
				if _, ok2 := eligibleParents[candidate.Pid()]; !ok2 {
					// A new descendant.
					eligibleParents[candidate.Pid()] = struct{}{}
				}
			}
		}

		// Scan for any descendants that have gone away.
		// This is more reliable than checking whether the status can be read.
		for pid := range eligibleParents {
			_, ok := allProcsMap[pid]
			if !ok {
				delete(eligibleParents, pid)
			}
		}

		// Update the current memory used by self and each known descendant.
		allProcMem := uint64(0)
		for pid := range eligibleParents {

			procMem, err := getMemoryUseForPID(pid)
			if err != nil {

				// Must check whether process still exists.
				allProcs, err := ps.Processes()
				if err != nil {
					mm.logError(err)
					// Don't log the same error many times over.
					break
				}

				processStillExists := false
				for _, oneProc := range allProcs {
					if oneProc.Pid() == pid {
						processStillExists = true
					}
				}

				// If the process still exists, record the error.  If the process is gone, ignore the error.
				if processStillExists {
					mm.logError(fmt.Errorf("failed to get memory use value for PID: %d", pid))
					// Don't log the same error many times over.
					break
				}

				// Don't use procMem if there was an error trying to get it.
				continue
			}

			allProcMem += procMem

			// Warn if we're close to the limit.
			if (warnMemoryUseAt != 0) && (allProcMem >= warnMemoryUseAt) {
				mm.jobLogger.Errorf("WARNING: Job has exceeded %d%% of %s memory limit",
					mm.limitBytesToPercent(warnMemoryUseAt), humanize.IBytes(mm.memoryLimit))

				// Now, look for the higher threshold unless we're already over it.
				warnMemoryUseAt = warnMemoryUseAt2
				if allProcMem >= warnMemoryUseAt {
					// Already over the higher threshold, so no more warnings.
					warnMemoryUseAt = 0
				}
			}
		}

		// Could capture the current process's memory use via runtime.ReadMemStats and memstats.Sys.
		// However, for consistency between all the processes, it is captured above.

		// Check the execute-done channel here.
		// This check must be non-blocking.
		exitMainLoop := false
		select {
		case <-mm.doneChannel:
			exitMainLoop = true // exit the main loop
		default:
			// Do nothing here.  Nothing has been sent or the channel has been closed.
		}

		if exitMainLoop {
			break
		}
	}
}

func (mm *memoryMonitor) limitFractionToBytes(fraction float64) uint64 {
	return uint64(fraction * float64(mm.memoryLimit))
}

func (mm *memoryMonitor) limitBytesToPercent(limitBytes uint64) int {
	return int(math.Round(100.0 * float64(limitBytes) / float64(mm.memoryLimit)))
}

func (mm *memoryMonitor) logError(e error) {
	mm.jobLogger.Errorf("memory monitor failed: %v", e)
}

func getMemoryUseForPID(pid int) (uint64, error) {

	proc, err := procfs.NewProc(pid)
	if err != nil {
		return uint64(0), err
	}

	procStatus, err := proc.NewStatus()
	if err != nil {
		return uint64(0), err
	}

	return procStatus.VmRSS, nil
}

// The End.
