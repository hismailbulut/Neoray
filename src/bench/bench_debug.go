//go:build debug
// +build debug

package bench

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hismailbulut/neoray/src/logger"
	tw "github.com/olekukonko/tablewriter"
)

const BUILD_TYPE = logger.DebugBuild

func IsDebugBuild() bool { return true }

var cpuProfileFile *os.File

// NOTE: All functions, types, constants and variables must exist in utils_release.go,
// Add empty ones to release file if they are not local.

func ToggleCpuProfile() error {
	if cpuProfileFile == nil {
		// New profile
		var err error
		cpuProfileFile, err = os.Create("Neoray_cpu_profile.prof")
		if err != nil {
			return fmt.Errorf("Failed to open cpu profile file: %s", err)
		}
		err = pprof.StartCPUProfile(cpuProfileFile)
		if err != nil {
			logger.Log(logger.ERROR)
			cpuProfileFile.Close()
			cpuProfileFile = nil
			return fmt.Errorf("Failed to start cpu profile: %s", err)
		}
		logger.Log(logger.DEBUG, "Profiling CPU usage...")
	} else {
		// Finish current profile
		pprof.StopCPUProfile()
		cpuProfileFile.Close()
		cpuProfileFile = nil
		logger.Log(logger.DEBUG, "CPU profile stopped and dumped to file")
	}
	return nil
}

func DumpHeapProfile() error {
	heapFile, err := os.Create("Neoray_heap_profile.prof")
	if err != nil {
		return fmt.Errorf("Failed to open memory profile file: %s", err)
	}
	defer heapFile.Close()
	runtime.GC()
	err = pprof.Lookup("heap").WriteTo(heapFile, 0)
	if err != nil {
		return fmt.Errorf("Failed to write memory profile: %s", err)
	}
	logger.Log(logger.DEBUG, "Heap profile dumped to file")
	return nil
}

type byteFormat float64

func (bytes byteFormat) String() string {
	n := bytes
	t := "b"
	switch {
	case bytes >= 1024*1024*1024:
		n = bytes / (1024 * 1024 * 1024)
		t = "gib"
	case bytes >= 1024*1024:
		n = bytes / (1024 * 1024)
		t = "mib"
	case bytes >= 1024:
		n = bytes / 1024
		t = "kib"
	}
	return fmt.Sprintf("%.2f %s", n, t)
}

type function_measure struct {
	calls int
	// CPU
	totalTime time.Duration
	maxTime   time.Duration
	// MEMORY
	totalAlloc  byteFormat
	maxAlloc    byteFormat
	totalMalloc int
	maxMalloc   int
}

var (
	mutex    sync.Mutex
	averages map[string]function_measure
	initTime time.Time
)

func init() {
	mutex.Lock()
	defer mutex.Unlock()
	averages = make(map[string]function_measure)
	initTime = time.Now()
}

// Runtime benchmark, dont use in production because function itself is costly
func BeginBenchmark() (EndBenchmark func(name string)) {
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	before := time.Now()
	// This funtion must called after bencmark
	EndBenchmark = func(name string) {
		// This function is costly
		elapsed := time.Since(before)
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		mutex.Lock()
		defer mutex.Unlock()
		val, ok := averages[name]
		if ok {
			val.calls++
			// CPU
			val.totalTime += elapsed
			if elapsed > val.maxTime {
				val.maxTime = elapsed
			}
			// MEMORY
			totalAlloc := byteFormat(m2.TotalAlloc - m1.TotalAlloc)
			val.totalAlloc += totalAlloc
			if totalAlloc > val.maxAlloc {
				val.maxAlloc = totalAlloc
			}
			mallocs := int(m2.Mallocs - m1.Mallocs)
			val.totalMalloc += mallocs
			if mallocs > val.maxMalloc {
				val.maxMalloc = mallocs
			}
			averages[name] = val
		} else {
			averages[name] = function_measure{
				calls:       1,
				totalTime:   elapsed,
				maxTime:     elapsed,
				totalAlloc:  byteFormat(m2.TotalAlloc - m1.TotalAlloc),
				totalMalloc: int(m2.Mallocs - m1.Mallocs),
			}
		}
	}
	return
}

// This prints a table which has the measurement information of all functions.
func PrintResults() {
	mutex.Lock()
	defer mutex.Unlock()

	if len(averages) == 0 {
		return
	}

	type funcTimeSummary struct {
		name  string
		calls int
		// CPU
		time    time.Duration
		average time.Duration
		max     time.Duration
		percent float64
		// ALLOC
		totalAlloc    byteFormat
		avgTotalAlloc byteFormat
		maxTotalAlloc byteFormat
		// MALLOC
		mallocs    int
		avgMallocs float64
		maxMallocs int
	}

	elapsedSeconds := time.Since(initTime).Seconds()

	sorted := []funcTimeSummary{}
	for key, val := range averages {
		sum := funcTimeSummary{}
		sum.name = strings.Replace(key, "main.", "", 1)
		sum.calls = val.calls
		// CPU
		sum.time = val.totalTime
		sum.average = val.totalTime / time.Duration(val.calls)
		sum.max = val.maxTime
		sum.percent = sum.time.Seconds() / elapsedSeconds
		// MEMORY
		sum.totalAlloc = val.totalAlloc
		sum.avgTotalAlloc = val.totalAlloc / byteFormat(val.calls)
		sum.maxTotalAlloc = val.maxAlloc

		sum.mallocs = val.totalMalloc
		sum.avgMallocs = float64(val.totalMalloc) / float64(val.calls)
		sum.maxMallocs = val.maxMalloc

		sorted = append(sorted, sum)
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].time > sorted[j].time
	})

	table := tw.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetAlignment(tw.ALIGN_CENTER)
	table.SetHeader([]string{
		"NAME", "CALLS", "PERCENT",
		"CPU(TOTAL - AVG - MAX)",
		"ALLOC(TOTAL - AVG - MAX)",
		"MALLOC(TOTAL - AVG- MAX)",
	})
	for _, r := range sorted {
		table.Append([]string{
			r.name,
			fmt.Sprintf("%d", r.calls),
			fmt.Sprintf("%.4f", r.percent), // This is also cpu
			// CPU
			fmt.Sprintf("%v - %v - %v", r.time, r.average, r.max),
			// ALLOC
			fmt.Sprintf("%v - %v - %v", r.totalAlloc, r.avgTotalAlloc, r.maxTotalAlloc),
			// MALLOC
			fmt.Sprintf("%d - %.2f - %d", r.mallocs, r.avgMallocs, r.maxMallocs),
		})
	}
	table.Render()
}
