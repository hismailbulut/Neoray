//go:build debug
// +build debug

package bench

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/hismailbulut/Neoray/pkg/logger"
	"github.com/olekukonko/tablewriter"
)

const BUILD_TYPE = logger.DebugBuild

func IsDebugBuild() bool { return true }

type funcBenchmark struct {
	calls     int
	totalTime time.Duration
	maxTime   time.Duration
}

var (
	mutex    sync.Mutex
	averages map[string]funcBenchmark
	initTime time.Time
)

func init() {
	mutex.Lock()
	defer mutex.Unlock()
	averages = make(map[string]funcBenchmark)
	initTime = time.Now()
}

// Runtime benchmark, dont use in production because function itself is costly
func Begin() func(name ...string) {
	before := time.Now()
	return func(name ...string) {
		elapsed := time.Since(before)
		mutex.Lock()
		defer mutex.Unlock()
		benchName := "unknown"
		if len(name) > 0 {
			benchName = name[0]
		} else {
			// get caller function name
			pc, _, _, ok := runtime.Caller(1)
			if ok {
				benchName = runtime.FuncForPC(pc).Name()
			}
		}
		val, ok := averages[benchName]
		if ok {
			val.calls++
			val.totalTime += elapsed
			if elapsed > val.maxTime {
				val.maxTime = elapsed
			}
			averages[benchName] = val
		} else {
			averages[benchName] = funcBenchmark{
				calls:     1,
				totalTime: elapsed,
				maxTime:   elapsed,
			}
		}
	}
}

// This prints a table which has the measurement information of all functions.
func PrintResults(out io.Writer) {
	mutex.Lock()
	defer mutex.Unlock()

	if len(averages) == 0 {
		return
	}

	type funcBenchmarkSummary struct {
		name    string
		calls   int
		time    time.Duration
		average time.Duration
		max     time.Duration
		percent float64
	}

	elapsedSeconds := time.Since(initTime).Seconds()

	list := []funcBenchmarkSummary{}
	for key, val := range averages {
		sum := funcBenchmarkSummary{}
		sum.name = key
		sum.calls = val.calls
		sum.time = val.totalTime
		sum.average = val.totalTime / time.Duration(val.calls)
		sum.max = val.maxTime
		sum.percent = sum.time.Seconds() / elapsedSeconds
		list = append(list, sum)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].time > list[j].time
	})

	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetHeader([]string{
		"NAME", "CALLS", "PERCENT",
		"TOTAL", "AVERAGE", "MAX",
	})
	for _, r := range list {
		table.Append([]string{
			r.name,
			fmt.Sprintf("%d", r.calls),
			fmt.Sprintf("%.4f", r.percent),
			r.time.String(),
			r.average.String(),
			r.max.String(),
		})
	}
	table.Render()
	io.Copy(out, &buf)
}
